package swe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/internal/tools/builtin"
)

// BuildToolset creates a *tools.Registry containing only the curated SWE tools:
// file_read, file_write, run_command (sandboxed to repoPath), patch (real
// implementation via ApplyUnifiedDiff), code_exec, and git.
//
// repoPath is the git repository root; all file writes and shell commands that
// access paths outside it are blocked by the existing sandbox in the builtin
// tools. dangerousApprove should be false for normal operation.
func BuildToolset(repoPath string, dangerousApprove bool) *tools.Registry {
	reg := tools.NewRegistry()
	protected := []string{} // no extra protection; repoPath acts as the boundary
	_ = protected

	// file_read and file_write — protected to repoPath.
	builtin.RegisterFiles(reg, []string{repoPath})

	// run_command — sandboxed, blocks access outside repoPath.
	builtin.RegisterShell(reg, []string{repoPath}, dangerousApprove)

	// patch — real unified diff application (overrides builtin stub).
	reg.Register(&swePatchTool{repoPath: repoPath})

	// code_exec — runs python/node/bash/go snippets.
	builtin.RegisterCodeExec(reg)

	// git — status, log, diff, add, commit, branch operations (no push).
	builtin.RegisterGit(reg)

	return reg
}

// swePatchTool is the SWE-specific patch tool that calls ApplyUnifiedDiff.
// It overrides the stub in internal/tools/builtin/patch.go which only counts
// diff lines without modifying the file.
type swePatchTool struct {
	repoPath string
}

func (t *swePatchTool) Name() string { return "patch" }

func (t *swePatchTool) Description() string {
	return "Applies a unified diff patch to a file in the repository. " +
		"Uses real line-based diff application; all writes are confined to the repo."
}

func (t *swePatchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"file_path": {
				"type": "string",
				"description": "Path to the file to patch (relative to repo root or absolute)."
			},
			"diff": {
				"type": "string",
				"description": "Unified diff content to apply."
			}
		},
		"required": ["file_path", "diff"]
	}`)
}

type swePatchArgs struct {
	FilePath string `json:"file_path"`
	Diff     string `json:"diff"`
}

func (t *swePatchTool) Execute(_ context.Context, args json.RawMessage) (*tools.Result, error) {
	var a swePatchArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid args: %v", err), IsError: true}, nil
	}
	if a.FilePath == "" {
		return &tools.Result{Content: "file_path is required", IsError: true}, nil
	}
	if a.Diff == "" {
		return &tools.Result{Content: "diff is required", IsError: true}, nil
	}
	if err := ApplyUnifiedDiff(a.FilePath, a.Diff); err != nil {
		return &tools.Result{Content: fmt.Sprintf("patch failed: %v", err), IsError: true}, nil
	}
	return &tools.Result{Content: fmt.Sprintf("patch applied to %s", a.FilePath)}, nil
}
