package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/tools"
)

// CustomTool is a user-defined tool loaded from config.
// Supports 3 types: "command" (shell), "http" (API call), "script" (inline script).
type CustomTool struct {
	cfg    config.CustomToolConfig
	client *http.Client
}

func (t *CustomTool) Name() string        { return t.cfg.Name }
func (t *CustomTool) Description() string { return t.cfg.Description }

func (t *CustomTool) Parameters() json.RawMessage {
	props := map[string]any{}
	required := []string{}

	for _, p := range t.cfg.Parameters {
		props[p.Name] = map[string]any{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	data, _ := json.Marshal(schema)
	return data
}

func (t *CustomTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params map[string]any
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	timeout := time.Duration(t.cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	switch t.cfg.Type {
	case "command":
		return t.executeCommand(ctx, params, timeout)
	case "http":
		return t.executeHTTP(ctx, params, timeout)
	case "script":
		return t.executeScript(ctx, params, timeout)
	default:
		return &tools.Result{Content: fmt.Sprintf("unknown tool type: %s", t.cfg.Type), IsError: true}, nil
	}
}

func (t *CustomTool) executeCommand(ctx context.Context, params map[string]any, timeout time.Duration) (*tools.Result, error) {
	cmdStr := t.cfg.Command

	// Replace {{param}} placeholders with actual values.
	for k, v := range params {
		cmdStr = strings.ReplaceAll(cmdStr, "{{"+k+"}}", fmt.Sprintf("%v", v))
	}

	// Guard dangerous patterns before executing.
	if result := checkCommandSafety(cmdStr, nil); result != nil {
		return result, nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", cmdStr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &tools.Result{Content: fmt.Sprintf("command failed: %s", strings.TrimSpace(errMsg)), IsError: true}, nil
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = "(no output)"
	}
	if len(output) > 50000 {
		output = output[:50000] + "\n... (truncated)"
	}
	return &tools.Result{Content: output}, nil
}

func (t *CustomTool) executeHTTP(ctx context.Context, params map[string]any, timeout time.Duration) (*tools.Result, error) {
	method := t.cfg.Method
	if method == "" {
		method = "GET"
	}

	url := t.cfg.URL
	for k, v := range params {
		url = strings.ReplaceAll(url, "{{"+k+"}}", fmt.Sprintf("%v", v))
	}

	var bodyReader io.Reader
	if method != "GET" && method != "HEAD" {
		data, _ := json.Marshal(params)
		bodyReader = bytes.NewReader(data)
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("create request: %v", err), IsError: true}, nil
	}

	for k, v := range t.cfg.Headers {
		req.Header.Set(k, v)
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("request failed: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)
	if len(content) > 50000 {
		content = content[:50000] + "\n... (truncated)"
	}

	if resp.StatusCode >= 400 {
		return &tools.Result{Content: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, content), IsError: true}, nil
	}

	return &tools.Result{Content: content}, nil
}

func (t *CustomTool) executeScript(ctx context.Context, params map[string]any, timeout time.Duration) (*tools.Result, error) {
	// Script type: command contains inline script content.
	script := t.cfg.Command
	for k, v := range params {
		script = strings.ReplaceAll(script, "{{"+k+"}}", fmt.Sprintf("%v", v))
	}

	// Guard dangerous patterns before executing.
	if result := checkCommandSafety(script, nil); result != nil {
		return result, nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &tools.Result{Content: fmt.Sprintf("script failed: %s", strings.TrimSpace(errMsg)), IsError: true}, nil
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = "(no output)"
	}
	return &tools.Result{Content: output}, nil
}

// RegisterCustomTools loads user-defined tools from config and registers them.
func RegisterCustomTools(registry *tools.Registry, cfgTools []config.CustomToolConfig) {
	client := &http.Client{Timeout: 30 * time.Second}
	for _, cfg := range cfgTools {
		if cfg.Name == "" || cfg.Type == "" {
			continue
		}
		registry.Register(&CustomTool{cfg: cfg, client: client})
	}
}
