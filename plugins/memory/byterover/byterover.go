// Package byterover implements the ByteRover memory provider plugin.
//
// ByteRover provides persistent memory via the brv CLI, organizing knowledge
// into a hierarchical context tree with tiered retrieval (fuzzy text followed
// by LLM-driven search). Local-first with optional cloud sync.
//
// Requires: brv CLI installed (npm install -g byterover-cli or
// curl -fsSL https://byterover.dev/install.sh | sh).
//
// Config via environment variables:
//
//	BRV_API_KEY - ByteRover API key (for cloud features, optional for local)
package byterover

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/paths"
	"github.com/sadewadee/hera/plugins"
)

const (
	queryTimeout  = 10 * time.Second
	curateTimeout = 120 * time.Second
	minQueryLen   = 10
	minOutputLen  = 20
	maxResultLen  = 8000
)

var (
	brvPathLock   sync.Mutex
	cachedBrvPath string
	brvResolved   bool
)

// Provider implements plugins.MemoryProvider for ByteRover.
type Provider struct {
	cwd       string
	sessionID string
	turnCount int
	mu        sync.Mutex
}

// New creates a new ByteRover memory provider.
func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string { return "byterover" }

func (p *Provider) IsAvailable() bool {
	return resolveBrvPath() != ""
}

func (p *Provider) Initialize(sessionID string) error {
	p.cwd = filepath.Join(paths.HeraHome(), "byterover")
	p.sessionID = sessionID
	p.turnCount = 0
	if err := os.MkdirAll(p.cwd, 0o755); err != nil {
		return fmt.Errorf("creating byterover directory: %w", err)
	}
	return nil
}

func (p *Provider) SystemPromptBlock() string {
	if resolveBrvPath() == "" {
		return ""
	}
	return "# ByteRover Memory\n" +
		"Active. Persistent knowledge tree with hierarchical context.\n" +
		"Use brv_query to search past knowledge, brv_curate to store " +
		"important facts, brv_status to check state."
}

func (p *Provider) Prefetch(query, sessionID string) string {
	if len(strings.TrimSpace(query)) < minQueryLen {
		return ""
	}
	q := strings.TrimSpace(query)
	if len(q) > 5000 {
		q = q[:5000]
	}
	result := runBrv([]string{"query", "--", q}, queryTimeout, p.cwd)
	if result.Success && len(strings.TrimSpace(result.Output)) > minOutputLen {
		return "## ByteRover Context\n" + strings.TrimSpace(result.Output)
	}
	return ""
}

func (p *Provider) SyncTurn(userContent, assistantContent, sessionID string) {
	p.mu.Lock()
	p.turnCount++
	p.mu.Unlock()

	if len(strings.TrimSpace(userContent)) < minQueryLen {
		return
	}

	go func() {
		combined := fmt.Sprintf("User: %.2000s\nAssistant: %.2000s", userContent, assistantContent)
		result := runBrv([]string{"curate", "--", combined}, curateTimeout, p.cwd)
		if !result.Success {
			slog.Debug("ByteRover sync failed", "error", result.Error)
		}
	}()
}

func (p *Provider) OnMemoryWrite(action, target, content string) {
	if (action != "add" && action != "replace") || content == "" {
		return
	}
	go func() {
		label := "Agent memory"
		if target == "user" {
			label = "User profile"
		}
		runBrv([]string{"curate", "--", fmt.Sprintf("[%s] %s", label, content)}, curateTimeout, p.cwd)
	}()
}

func (p *Provider) OnPreCompress(messages []map[string]interface{}) string {
	if len(messages) == 0 {
		return ""
	}

	var parts []string
	start := 0
	if len(messages) > 10 {
		start = len(messages) - 10
	}
	for _, msg := range messages[start:] {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		if content != "" && (role == "user" || role == "assistant") {
			text := content
			if len(text) > 500 {
				text = text[:500]
			}
			parts = append(parts, fmt.Sprintf("%s: %s", role, text))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	combined := strings.Join(parts, "\n")
	go func() {
		result := runBrv([]string{"curate", "--", "[Pre-compression context]\n" + combined}, curateTimeout, p.cwd)
		if result.Success {
			slog.Info("ByteRover pre-compression flush", "messages", len(parts))
		}
	}()
	return ""
}

func (p *Provider) OnSessionEnd(_ []map[string]interface{}) {}

func (p *Provider) GetToolSchemas() []plugins.ToolSchema {
	return []plugins.ToolSchema{
		{
			Name: "brv_query",
			Description: "Search ByteRover's persistent knowledge tree for relevant context. " +
				"Returns memories, project knowledge, architectural decisions, and " +
				"patterns from previous sessions.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "What to search for.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name: "brv_curate",
			Description: "Store important information in ByteRover's persistent knowledge tree. " +
				"Use for architectural decisions, bug fixes, user preferences, project patterns.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The information to remember.",
					},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "brv_status",
			Description: "Check ByteRover status -- CLI version, context tree stats, cloud sync state.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
	}
}

func (p *Provider) HandleToolCall(toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "brv_query":
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query is required")
		}
		q := strings.TrimSpace(query)
		if len(q) > 5000 {
			q = q[:5000]
		}
		result := runBrv([]string{"query", "--", q}, queryTimeout, p.cwd)
		if !result.Success {
			return "", fmt.Errorf("%s", result.Error)
		}
		output := strings.TrimSpace(result.Output)
		if len(output) < minOutputLen {
			return jsonMarshal(map[string]string{"result": "No relevant memories found."}), nil
		}
		if len(output) > maxResultLen {
			output = output[:maxResultLen] + "\n\n[... truncated]"
		}
		return jsonMarshal(map[string]string{"result": output}), nil

	case "brv_curate":
		content, _ := args["content"].(string)
		if content == "" {
			return "", fmt.Errorf("content is required")
		}
		result := runBrv([]string{"curate", "--", content}, curateTimeout, p.cwd)
		if !result.Success {
			return "", fmt.Errorf("%s", result.Error)
		}
		return jsonMarshal(map[string]string{"result": "Memory curated successfully."}), nil

	case "brv_status":
		result := runBrv([]string{"status"}, 15*time.Second, p.cwd)
		if !result.Success {
			return "", fmt.Errorf("%s", result.Error)
		}
		return jsonMarshal(map[string]string{"status": result.Output}), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (p *Provider) GetConfigSchema() []plugins.ConfigField {
	return []plugins.ConfigField{
		{
			Key:         "api_key",
			Description: "ByteRover API key (optional, for cloud sync)",
			Secret:      true,
			EnvVar:      "BRV_API_KEY",
			URL:         "https://app.byterover.dev",
		},
	}
}

func (p *Provider) Shutdown() {}

// --- Helpers ---

type brvResult struct {
	Success bool
	Output  string
	Error   string
}

func resolveBrvPath() string {
	brvPathLock.Lock()
	defer brvPathLock.Unlock()

	if brvResolved {
		return cachedBrvPath
	}

	path, err := exec.LookPath("brv")
	if err == nil {
		cachedBrvPath = path
		brvResolved = true
		return path
	}

	homeDir, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(homeDir, ".brv-cli", "bin", "brv"),
		"/usr/local/bin/brv",
		filepath.Join(homeDir, ".npm-global", "bin", "brv"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			cachedBrvPath = c
			brvResolved = true
			return c
		}
	}

	cachedBrvPath = ""
	brvResolved = true
	return ""
}

func runBrv(args []string, timeout time.Duration, cwd string) brvResult {
	brvPath := resolveBrvPath()
	if brvPath == "" {
		return brvResult{Error: "brv CLI not found. Install: npm install -g byterover-cli"}
	}

	if cwd != "" {
		os.MkdirAll(cwd, 0o755)
	}

	cmdArgs := append([]string{brvPath}, args...)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(brvPath)+string(os.PathListSeparator)+os.Getenv("PATH"))

	done := make(chan struct{})
	var stdout, stderr []byte
	var cmdErr error

	go func() {
		stdout, cmdErr = cmd.Output()
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			stderr = exitErr.Stderr
		}
		close(done)
	}()

	select {
	case <-done:
		if cmdErr != nil {
			errMsg := strings.TrimSpace(string(stderr))
			if errMsg == "" {
				errMsg = strings.TrimSpace(string(stdout))
			}
			if errMsg == "" {
				errMsg = fmt.Sprintf("brv exited with error: %v", cmdErr)
			}
			return brvResult{Error: errMsg}
		}
		return brvResult{Success: true, Output: strings.TrimSpace(string(stdout))}
	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return brvResult{Error: fmt.Sprintf("brv timed out after %v", timeout)}
	}
}

func jsonMarshal(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
