package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"

	"github.com/sadewadee/hera/internal/tools"
)

type TerminalTool struct {
	mu       sync.Mutex
	sessions map[string]*termSession
}

type termSession struct {
	ID      string
	WorkDir string
	Env     []string
}

type terminalArgs struct {
	Command   string `json:"command"`
	SessionID string `json:"session_id,omitempty"`
	WorkDir   string `json:"work_dir,omitempty"`
}

func (t *TerminalTool) Name() string { return "terminal" }
func (t *TerminalTool) Description() string {
	return "Runs commands in a persistent terminal session with working directory and environment tracking."
}
func (t *TerminalTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":"Command to execute"},"session_id":{"type":"string","description":"Terminal session ID (creates new if omitted)"},"work_dir":{"type":"string","description":"Working directory"}},"required":["command"]}`)
}

func (t *TerminalTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a terminalArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	t.mu.Lock()
	if t.sessions == nil {
		t.sessions = make(map[string]*termSession)
	}
	sess, ok := t.sessions[a.SessionID]
	if !ok && a.SessionID != "" {
		sess = &termSession{ID: a.SessionID, WorkDir: a.WorkDir}
		t.sessions[a.SessionID] = sess
	}
	t.mu.Unlock()
	if result := checkCommandSafety(a.Command, nil); result != nil {
		return result, nil
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", a.Command)
	if sess != nil && sess.WorkDir != "" {
		cmd.Dir = sess.WorkDir
	} else if a.WorkDir != "" {
		cmd.Dir = a.WorkDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR: " + stderr.String()
	}
	if err != nil {
		output += fmt.Sprintf("\nExit error: %v", err)
	}
	if len(output) > 50000 {
		output = output[:50000] + "\n... [truncated]"
	}
	return &tools.Result{Content: output}, nil
}

func RegisterTerminal(registry *tools.Registry) { registry.Register(&TerminalTool{}) }
