// Package environments provides execution environment implementations.
//
// managed_modal.go implements a managed Modal environment backed by a
// tool gateway. The gateway owns the Modal sandbox lifecycle, and this
// client handles command execution, polling, and cleanup via HTTP.
package environments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	managedModalConnectTimeout = 1 * time.Second
	managedModalPollTimeout    = 5 * time.Second
	managedModalCancelTimeout  = 5 * time.Second
	managedModalPollInterval   = 250 * time.Millisecond
	managedModalGraceSeconds   = 10.0
)

// ManagedModalEnvironment is a gateway-owned Modal sandbox with
// Hera-compatible execute/cleanup.
type ManagedModalEnvironment struct {
	gatewayOrigin  string
	nousUserToken  string
	taskID         string
	persistent     bool
	image          string
	sandboxID      string
	cwd            string
	timeout        int
	sandboxKwargs  map[string]any
	idempotencyKey string
	httpClient     *http.Client
}

// ManagedModalConfig holds configuration for creating a managed Modal environment.
type ManagedModalConfig struct {
	Image                string
	CWD                  string
	Timeout              int
	SandboxKwargs        map[string]any
	PersistentFilesystem bool
	TaskID               string
	GatewayOrigin        string
	NousUserToken        string
}

// NewManagedModalEnvironment creates a managed Modal environment.
func NewManagedModalEnvironment(cfg ManagedModalConfig) (*ManagedModalEnvironment, error) {
	if cfg.GatewayOrigin == "" || cfg.NousUserToken == "" {
		return nil, fmt.Errorf("managed Modal requires a configured tool gateway and Nous user token")
	}

	if cfg.CWD == "" {
		cfg.CWD = "/root"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60
	}
	if cfg.TaskID == "" {
		cfg.TaskID = "default"
	}

	env := &ManagedModalEnvironment{
		gatewayOrigin:  cfg.GatewayOrigin,
		nousUserToken:  cfg.NousUserToken,
		taskID:         cfg.TaskID,
		persistent:     cfg.PersistentFilesystem,
		image:          cfg.Image,
		cwd:            cfg.CWD,
		timeout:        cfg.Timeout,
		sandboxKwargs:  cfg.SandboxKwargs,
		idempotencyKey: uuid.New().String(),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	sandboxID, err := env.createSandbox()
	if err != nil {
		return nil, err
	}
	env.sandboxID = sandboxID
	return env, nil
}

func (e *ManagedModalEnvironment) Name() string { return "managed_modal" }

// Execute runs a command in the managed Modal sandbox.
func (e *ManagedModalEnvironment) Execute(ctx context.Context, command string, args []string) (*ExecResult, error) {
	execID := uuid.New().String()
	fullCommand := command
	if len(args) > 0 {
		for _, a := range args {
			fullCommand += " " + a
		}
	}

	payload := map[string]any{
		"execId":    execID,
		"command":   fullCommand,
		"cwd":       e.cwd,
		"timeoutMs": e.timeout * 1000,
	}

	resp, err := e.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/sandboxes/%s/execs", e.sandboxID), payload)
	if err != nil {
		return &ExecResult{Stderr: fmt.Sprintf("managed Modal exec failed: %v", err), ExitCode: 1}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return &ExecResult{Stderr: formatGatewayError("managed Modal exec failed", resp.StatusCode, body), ExitCode: 1}, nil
	}

	var startResp struct {
		ExecID     string `json:"execId"`
		Status     string `json:"status"`
		Output     string `json:"output"`
		ReturnCode int    `json:"returncode"`
	}
	json.Unmarshal(body, &startResp)

	// If immediately completed.
	if isTerminalStatus(startResp.Status) {
		return &ExecResult{Stdout: startResp.Output, ExitCode: startResp.ReturnCode}, nil
	}

	// Poll for completion.
	deadline := time.Now().Add(time.Duration(e.timeout)*time.Second + time.Duration(managedModalGraceSeconds)*time.Second)
	for {
		select {
		case <-ctx.Done():
			e.cancelExec(execID)
			return &ExecResult{Stderr: "[Command interrupted - Modal sandbox exec cancelled]", ExitCode: 130}, nil
		default:
		}

		if time.Now().After(deadline) {
			e.cancelExec(execID)
			return &ExecResult{
				Stderr:   fmt.Sprintf("managed Modal exec timed out after %ds", e.timeout),
				ExitCode: 124,
			}, nil
		}

		result, done := e.pollExec(ctx, execID)
		if done {
			return result, nil
		}

		time.Sleep(managedModalPollInterval)
	}
}

func (e *ManagedModalEnvironment) ReadFile(_ context.Context, _ string) ([]byte, error) {
	return nil, fmt.Errorf("managed Modal does not support direct file reads")
}

func (e *ManagedModalEnvironment) WriteFile(_ context.Context, _ string, _ []byte) error {
	return fmt.Errorf("managed Modal does not support direct file writes")
}

// Cleanup terminates the managed Modal sandbox.
func (e *ManagedModalEnvironment) Cleanup(ctx context.Context) error {
	if e.sandboxID == "" {
		return nil
	}

	payload := map[string]any{
		"snapshotBeforeTerminate": e.persistent,
	}
	resp, err := e.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/sandboxes/%s/terminate", e.sandboxID), payload)
	if err != nil {
		slog.Warn("managed Modal cleanup failed", "error", err)
		return err
	}
	resp.Body.Close()
	e.sandboxID = ""
	return nil
}

func (e *ManagedModalEnvironment) createSandbox() (string, error) {
	cpu := coerceNumber(e.sandboxKwargs["cpu"], 1)
	memory := coerceNumber(firstOf(e.sandboxKwargs["memoryMiB"], e.sandboxKwargs["memory"]), 5120)

	createPayload := map[string]any{
		"image":                e.image,
		"cwd":                  e.cwd,
		"cpu":                  cpu,
		"memoryMiB":            memory,
		"timeoutMs":            3_600_000,
		"idleTimeoutMs":        max(300_000, e.timeout*1000),
		"persistentFilesystem": e.persistent,
		"logicalKey":           e.taskID,
	}
	if disk := coerceNumber(firstOf(e.sandboxKwargs["ephemeral_disk"], e.sandboxKwargs["diskMiB"]), -1); disk > 0 {
		createPayload["diskMiB"] = disk
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := e.newRequest(ctx, http.MethodPost, "/v1/sandboxes", createPayload)
	if err != nil {
		return "", fmt.Errorf("managed Modal create failed: %w", err)
	}
	req.Header.Set("X-Idempotency-Key", e.idempotencyKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("managed Modal create failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("%s", formatGatewayError("managed Modal create failed", resp.StatusCode, body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.ID == "" {
		return "", fmt.Errorf("managed Modal create did not return a sandbox id")
	}
	return result.ID, nil
}

func (e *ManagedModalEnvironment) pollExec(ctx context.Context, execID string) (*ExecResult, bool) {
	resp, err := e.doRequest(ctx, http.MethodGet,
		fmt.Sprintf("/v1/sandboxes/%s/execs/%s", e.sandboxID, execID), nil)
	if err != nil {
		return &ExecResult{Stderr: fmt.Sprintf("managed Modal exec poll failed: %v", err), ExitCode: 1}, true
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		return &ExecResult{Stderr: "managed Modal exec not found", ExitCode: 1}, true
	}
	if resp.StatusCode >= 400 {
		return &ExecResult{Stderr: formatGatewayError("managed Modal exec poll failed", resp.StatusCode, body), ExitCode: 1}, true
	}

	var status struct {
		Status     string `json:"status"`
		Output     string `json:"output"`
		ReturnCode int    `json:"returncode"`
	}
	json.Unmarshal(body, &status)

	if isTerminalStatus(status.Status) {
		return &ExecResult{Stdout: status.Output, ExitCode: status.ReturnCode}, true
	}
	return nil, false
}

func (e *ManagedModalEnvironment) cancelExec(execID string) {
	ctx, cancel := context.WithTimeout(context.Background(), managedModalCancelTimeout)
	defer cancel()
	resp, err := e.doRequest(ctx, http.MethodPost,
		fmt.Sprintf("/v1/sandboxes/%s/execs/%s/cancel", e.sandboxID, execID), nil)
	if err != nil {
		slog.Warn("managed Modal exec cancel failed", "error", err)
		return
	}
	resp.Body.Close()
}

func (e *ManagedModalEnvironment) doRequest(ctx context.Context, method, path string, payload map[string]any) (*http.Response, error) {
	req, err := e.newRequest(ctx, method, path, payload)
	if err != nil {
		return nil, err
	}
	return e.httpClient.Do(req)
}

func (e *ManagedModalEnvironment) newRequest(ctx context.Context, method, path string, payload map[string]any) (*http.Request, error) {
	url := e.gatewayOrigin + path
	var bodyReader io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.nousUserToken)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func isTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled", "timeout":
		return true
	}
	return false
}

func formatGatewayError(prefix string, statusCode int, body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, key := range []string{"error", "message", "code"} {
			if msg, ok := payload[key].(string); ok && msg != "" {
				return fmt.Sprintf("%s: %s", prefix, msg)
			}
		}
	}
	text := string(body)
	if text != "" {
		return fmt.Sprintf("%s: %s", prefix, text)
	}
	return fmt.Sprintf("%s: HTTP %d", prefix, statusCode)
}

func coerceNumber(value any, defaultVal float64) float64 {
	if value == nil {
		return defaultVal
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func firstOf(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func requestTimeoutEnv(name string, defaultVal float64) float64 {
	v := os.Getenv(name)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return defaultVal
	}
	return f
}
