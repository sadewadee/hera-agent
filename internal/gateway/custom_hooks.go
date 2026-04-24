package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/config"
)

// CustomHook is a user-defined hook loaded from config.
// It runs a shell command or HTTP call on agent lifecycle events.
type CustomHook struct {
	cfg    config.HookConfig
	client *http.Client
}

func (h *CustomHook) Name() string { return h.cfg.Name }

func (h *CustomHook) BeforeMessage(ctx context.Context, msg *IncomingMessage) (*IncomingMessage, error) {
	if h.cfg.Event != "before_message" {
		return msg, nil
	}
	return msg, h.run(ctx, map[string]any{
		"event":    "before_message",
		"platform": msg.Platform,
		"user_id":  msg.UserID,
		"text":     msg.Text,
	})
}

func (h *CustomHook) AfterMessage(ctx context.Context, msg *IncomingMessage, response string) error {
	if h.cfg.Event != "after_message" {
		return nil
	}
	return h.run(ctx, map[string]any{
		"event":    "after_message",
		"platform": msg.Platform,
		"user_id":  msg.UserID,
		"text":     msg.Text,
		"response": response,
	})
}

func (h *CustomHook) run(ctx context.Context, data map[string]any) error {
	switch h.cfg.Type {
	case "command":
		return h.runCommand(ctx, data)
	case "http":
		return h.runHTTP(ctx, data)
	default:
		return fmt.Errorf("unknown hook type: %s", h.cfg.Type)
	}
}

func (h *CustomHook) runCommand(ctx context.Context, data map[string]any) error {
	cmdStr := h.cfg.Command

	// Inject data as env vars prefixed with HERA_HOOK_.
	env := make([]string, 0, len(data))
	for k, v := range data {
		env = append(env, fmt.Sprintf("HERA_HOOK_%s=%v", strings.ToUpper(k), v))
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", cmdStr)
	cmd.Env = append(cmd.Environ(), env...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hook %s command failed: %v (output: %s)", h.cfg.Name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (h *CustomHook) runHTTP(ctx context.Context, data map[string]any) error {
	body, _ := json.Marshal(data)

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, h.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("hook %s create request: %w", h.cfg.Name, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("hook %s request failed: %w", h.cfg.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("hook %s returned HTTP %d", h.cfg.Name, resp.StatusCode)
	}
	return nil
}

// RegisterCustomHooks loads user-defined hooks from config and registers them.
func RegisterCustomHooks(hm *HookManager, cfgHooks []config.HookConfig) {
	client := &http.Client{Timeout: 10 * time.Second}
	for _, cfg := range cfgHooks {
		if cfg.Name == "" || cfg.Event == "" || cfg.Type == "" {
			continue
		}
		hm.Register(&CustomHook{cfg: cfg, client: client})
	}
}
