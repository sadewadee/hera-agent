package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sadewadee/hera/internal/config"
)

// HooksWatcher watches a directory for YAML hook definitions and registers
// them into a HookManager at startup and whenever files change.
//
// Each YAML file in the directory must be a list of config.HookConfig objects:
//
//   - name: my-hook
//     event: before_message
//     type: command
//     command: echo "hello"
//
// Files are polled every 5 seconds. On change: old hooks from that file are
// removed by name and new ones are registered. Deleted files cause the
// hooks they defined to be removed.
//
// Poll-based watching avoids platform-specific bugs (macOS FSEvents
// coalescing, Linux inotify limits, container overlay filesystems) at the
// cost of up to 5 seconds of lag — acceptable for human-managed hook files.
type HooksWatcher struct {
	hm     *HookManager
	dir    string
	client *http.Client

	interval time.Duration
	logger   *slog.Logger

	mu     sync.Mutex
	hashes map[string]string   // file path -> sha256 of last parsed content
	byFile map[string][]string // file path -> hook names registered from it
	stopCh chan struct{}
	once   sync.Once
}

// NewHooksWatcher creates a watcher backed by the given HookManager.
// Call Start(ctx) to begin polling. Safe to create even when dir is empty.
func NewHooksWatcher(hm *HookManager, dir string) *HooksWatcher {
	return &HooksWatcher{
		hm:       hm,
		dir:      dir,
		client:   &http.Client{Timeout: 10 * time.Second},
		interval: 5 * time.Second,
		logger:   slog.Default(),
		hashes:   make(map[string]string),
		byFile:   make(map[string][]string),
		stopCh:   make(chan struct{}),
	}
}

// Start performs an immediate scan then begins the background polling loop.
// Non-blocking; the goroutine exits when ctx is cancelled or Stop is called.
func (w *HooksWatcher) Start(ctx context.Context) {
	_ = os.MkdirAll(w.dir, 0o755)
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

// Stop signals the polling goroutine to exit.
func (w *HooksWatcher) Stop() {
	w.once.Do(func() { close(w.stopCh) })
}

// scan reads all *.yaml / *.yml files in the watched directory. For each:
//   - Skip if content hash is unchanged.
//   - Remove previously registered hooks from that file by name.
//   - Parse and register the new/updated hook list.
//
// Deleted files have their hooks removed.
func (w *HooksWatcher) scan() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		if !os.IsNotExist(err) {
			w.logger.Warn("hooks-watcher: read dir", "dir", w.dir, "err", err)
		}
		return
	}

	seen := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isHookYAML(name) {
			continue
		}
		full := filepath.Join(w.dir, name)
		seen[full] = true

		data, err := os.ReadFile(full)
		if err != nil {
			w.logger.Warn("hooks-watcher: read file", "file", full, "err", err)
			continue
		}
		hash := sha256hex(data)

		w.mu.Lock()
		prev, exists := w.hashes[full]
		w.mu.Unlock()

		if exists && prev == hash {
			continue // unchanged
		}

		var cfgs []config.HookConfig
		if err := yaml.Unmarshal(data, &cfgs); err != nil {
			w.logger.Warn("hooks-watcher: parse failed", "file", full, "err", err)
			continue
		}

		// Remove stale hooks from a previous version of this file.
		w.mu.Lock()
		for _, hookName := range w.byFile[full] {
			w.hm.Unregister(hookName)
		}
		w.mu.Unlock()

		// Register new hooks and track their names for future removal.
		var registered []string
		for i := range cfgs {
			cfg := cfgs[i]
			if cfg.Name == "" || cfg.Event == "" || cfg.Type == "" {
				w.logger.Warn("hooks-watcher: skipping incomplete hook entry",
					"file", full, "index", i)
				continue
			}
			w.hm.Register(&CustomHook{cfg: cfg, client: w.client})
			registered = append(registered, cfg.Name)
		}

		w.mu.Lock()
		w.hashes[full] = hash
		w.byFile[full] = registered
		w.mu.Unlock()

		w.logger.Info("hooks-watcher: loaded",
			"file", full, "count", len(registered), "updated", exists)
	}

	// Handle deleted files: remove their hooks and clean up tracking state.
	w.mu.Lock()
	for path, hookNames := range w.byFile {
		if !seen[path] {
			for _, hookName := range hookNames {
				w.hm.Unregister(hookName)
			}
			delete(w.hashes, path)
			delete(w.byFile, path)
			w.logger.Info("hooks-watcher: removed hooks from deleted file",
				"file", path, "removed", hookNames)
		}
	}
	w.mu.Unlock()
}

func isHookYAML(name string) bool {
	if len(name) == 0 || name[0] == '.' {
		return false
	}
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

// sha256hex returns the lowercase hex SHA-256 of b.
// Defined here to avoid a cross-package dependency on builtin.sha256hex.
func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
