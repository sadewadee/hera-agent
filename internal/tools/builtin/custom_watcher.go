package builtin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/tools"
)

// DefaultCustomToolsDir is where hot-reloadable tool definitions live
// by convention: ~/.hera/tools.d/. Each file is a single YAML object
// matching config.CustomToolConfig. Poll-based watching keeps the
// project dep-free; 5-second interval is imperceptible for
// human/agent-driven tool creation and avoids platform-specific bugs
// (macOS FSEvents coalescing, Linux inotify limits, container overlay
// quirks).
func DefaultCustomToolsDir(heraDir string) string {
	return filepath.Join(heraDir, "tools.d")
}

// CustomToolWatcher owns the hot-reload loop for tools.d files. It
// tracks what is currently registered by file path so a deleted file
// removes the tool cleanly without leaking registrations.
type CustomToolWatcher struct {
	registry *tools.Registry
	dir      string
	interval time.Duration
	logger   *slog.Logger

	mu       sync.Mutex
	current  map[string]string // file path -> content hash
	tools    map[string]string // tool name -> file path (for cleanup)
	client   *http.Client
	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewCustomToolWatcher creates a watcher. Call Start to begin the
// polling loop. Safe on a nil registry (no-op).
func NewCustomToolWatcher(registry *tools.Registry, dir string) *CustomToolWatcher {
	return &CustomToolWatcher{
		registry: registry,
		dir:      dir,
		interval: 5 * time.Second,
		logger:   slog.Default(),
		current:  make(map[string]string),
		tools:    make(map[string]string),
		client:   &http.Client{Timeout: 30 * time.Second},
		stopCh:   make(chan struct{}),
	}
}

// Start kicks off the watcher goroutine. Non-blocking. Does one
// immediate scan so tools already in the directory at startup show up
// without waiting for the first tick.
func (w *CustomToolWatcher) Start(ctx context.Context) {
	if w.registry == nil || w.dir == "" {
		return
	}
	_ = os.MkdirAll(w.dir, 0o755)
	// Initial scan synchronously so callers see tools immediately.
	w.scan()

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.scan()
			}
		}
	}()
}

// Stop signals the goroutine to exit.
func (w *CustomToolWatcher) Stop() {
	w.stopOnce.Do(func() { close(w.stopCh) })
}

// scan reads every *.yaml file in dir, compares content hashes against
// last scan, and (re)registers or unregisters tools accordingly.
func (w *CustomToolWatcher) scan() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		// Directory may not exist yet — silently ignore on this pass.
		if !os.IsNotExist(err) {
			w.logger.Warn("custom-tools watcher: read dir", "dir", w.dir, "err", err)
		}
		return
	}

	seen := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isToolYAML(name) {
			continue
		}
		full := filepath.Join(w.dir, name)
		seen[full] = true

		data, err := os.ReadFile(full)
		if err != nil {
			w.logger.Warn("custom-tools watcher: read file", "file", full, "err", err)
			continue
		}
		hash := sha256hex(data)

		w.mu.Lock()
		prev, exists := w.current[full]
		w.mu.Unlock()

		if exists && prev == hash {
			continue // unchanged
		}

		var cfg config.CustomToolConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			w.logger.Warn("custom-tools watcher: parse", "file", full, "err", err)
			continue
		}
		if cfg.Name == "" || cfg.Type == "" {
			w.logger.Warn("custom-tools watcher: skipping incomplete tool", "file", full)
			continue
		}

		w.registry.Register(&CustomTool{cfg: cfg, client: w.client})

		w.mu.Lock()
		w.current[full] = hash
		w.tools[cfg.Name] = full
		w.mu.Unlock()

		w.logger.Info("custom-tools watcher: registered",
			"name", cfg.Name, "type", cfg.Type, "file", full,
			"updated", exists,
		)
	}

	// Detect deletions.
	w.mu.Lock()
	var removed []string
	for path, name := range w.pathToName() {
		if !seen[path] {
			removed = append(removed, name)
			delete(w.current, path)
			delete(w.tools, name)
		}
	}
	w.mu.Unlock()

	for _, name := range removed {
		if err := w.registry.Unregister(name); err != nil {
			// Tool may have already been superseded by a re-register; log and continue.
			w.logger.Warn("custom-tools watcher: unregister failed", "name", name, "err", err)
		} else {
			w.logger.Info("custom-tools watcher: tool deregistered", "name", name)
		}
	}
}

// pathToName inverts the tools map; caller must hold w.mu.
func (w *CustomToolWatcher) pathToName() map[string]string {
	out := make(map[string]string, len(w.tools))
	for name, path := range w.tools {
		out[path] = name
	}
	return out
}

func isToolYAML(name string) bool {
	if len(name) == 0 || name[0] == '.' {
		return false
	}
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// RegisterCustomToolTool registers a meta-tool that lets the agent
// add new CustomTool definitions at runtime by writing a YAML file
// into ~/.hera/tools.d/. The watcher picks them up on the next scan
// (5 seconds) so the LLM sees the new tool on subsequent turns.
func RegisterCustomToolTool(registry *tools.Registry, dir string) {
	registry.Register(&registerCustomToolTool{dir: dir})
}

// registerCustomToolTool is the agent-callable "write a yaml spec and
// drop it in tools.d so it becomes a new tool next turn" mechanism.
type registerCustomToolTool struct {
	dir string
}

func (t *registerCustomToolTool) Name() string { return "register_custom_tool" }

func (t *registerCustomToolTool) Description() string {
	return `Creates a new agent tool at runtime by writing a YAML spec to ~/.hera/tools.d/. The watcher registers it within ~5 seconds.

Tool types supported:
- "command": shell command with {{param}} interpolation
- "http":    HTTP request with URL, method, headers, and body
- "script":  inline shell script

The saved YAML mirrors config.CustomToolConfig. Parameters are declared so the LLM can call the tool with typed arguments later. Tool survives agent restarts because the YAML persists on disk.

Use when the user asks for a durable capability the agent doesn't yet have and the task fits a shell/HTTP wrapper. For tasks needing real Python logic, use python_register_tool (when the MCP python server is enabled) instead.`
}

func (t *registerCustomToolTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"name":        {"type": "string", "description": "File-safe slug [a-z0-9_-]{1,64}."},
			"description": {"type": "string", "description": "One-line summary shown to the LLM when choosing tools."},
			"type":        {"type": "string", "enum": ["command", "http", "script"]},
			"command":     {"type": "string", "description": "Shell command template for type=command. Supports {{param}} placeholders."},
			"url":         {"type": "string", "description": "URL for type=http."},
			"method":      {"type": "string", "description": "HTTP method (GET/POST/etc) for type=http. Defaults to GET."},
			"headers":     {"type": "object", "description": "HTTP headers for type=http."},
			"parameters": {
				"type": "array",
				"description": "Parameters the tool accepts. Each entry: {name, type, description, required}.",
				"items": {
					"type": "object",
					"properties": {
						"name":        {"type": "string"},
						"type":        {"type": "string", "enum": ["string", "integer", "boolean", "number"]},
						"description": {"type": "string"},
						"required":    {"type": "boolean"}
					},
					"required": ["name", "type"]
				}
			},
			"timeout":     {"type": "integer", "description": "Max execution time in seconds. Defaults to 30."}
		},
		"required": ["name", "description", "type"]
	}`)
}

func (t *registerCustomToolTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var spec config.CustomToolConfig
	if err := yaml.Unmarshal(coerceJSONtoYAML(args), &spec); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	name := strings.TrimSpace(spec.Name)
	if !nameRE.MatchString(name) {
		return &tools.Result{Content: "name must be lowercase alphanumeric plus dash/underscore, 1-64 chars", IsError: true}, nil
	}
	if spec.Type != "command" && spec.Type != "http" && spec.Type != "script" {
		return &tools.Result{Content: "type must be one of: command, http, script", IsError: true}, nil
	}
	if spec.Type == "command" && strings.TrimSpace(spec.Command) == "" {
		return &tools.Result{Content: "type=command requires a command template", IsError: true}, nil
	}
	if spec.Type == "http" && strings.TrimSpace(spec.URL) == "" {
		return &tools.Result{Content: "type=http requires url", IsError: true}, nil
	}
	if spec.Type == "script" && strings.TrimSpace(spec.Command) == "" {
		return &tools.Result{Content: "type=script requires command (the inline script body)", IsError: true}, nil
	}

	if err := os.MkdirAll(t.dir, 0o755); err != nil {
		return &tools.Result{Content: fmt.Sprintf("create tools dir: %v", err), IsError: true}, nil
	}
	out, err := yaml.Marshal(&spec)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("marshal yaml: %v", err), IsError: true}, nil
	}
	path := filepath.Join(t.dir, name+".yaml")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return &tools.Result{Content: fmt.Sprintf("write tool file: %v", err), IsError: true}, nil
	}
	return &tools.Result{
		Content: fmt.Sprintf("registered tool %q (type=%s) at %s. It will be callable within 5 seconds.", name, spec.Type, path),
	}, nil
}

// coerceJSONtoYAML converts the JSON bytes the tool receives into YAML
// the yaml.Unmarshal expects. The shapes overlap for this struct, so
// unmarshalling JSON as YAML works — yaml.v3 parses JSON as valid YAML.
func coerceJSONtoYAML(b []byte) []byte { return b }
