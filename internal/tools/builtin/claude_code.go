package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sadewadee/hera/internal/tools"
)

// ClaudeCodeTool spawns the `claude` CLI process to execute tasks.
type ClaudeCodeTool struct{}

type claudeCodeArgs struct {
	Prompt     string `json:"prompt"`
	WorkingDir string `json:"working_dir,omitempty"`
	MaxTokens  int    `json:"max_tokens,omitempty"`
}

func (c *ClaudeCodeTool) Name() string {
	return "claude_code"
}

func (c *ClaudeCodeTool) Description() string {
	return "Runs a prompt through the Claude Code CLI (claude command). Useful for code generation, analysis, and complex reasoning tasks that benefit from Claude's capabilities."
}

func (c *ClaudeCodeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"prompt": {
				"type": "string",
				"description": "The prompt/task to send to Claude Code."
			},
			"working_dir": {
				"type": "string",
				"description": "Working directory for the claude process. Defaults to current directory."
			},
			"max_tokens": {
				"type": "integer",
				"description": "Maximum output tokens. Defaults to 4096."
			}
		},
		"required": ["prompt"]
	}`)
}

func (c *ClaudeCodeTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params claudeCodeArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Prompt == "" {
		return &tools.Result{Content: "prompt is required", IsError: true}, nil
	}

	// Check if claude CLI is available.
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return &tools.Result{
			Content: "claude CLI not found. Install it from https://docs.anthropic.com/claude-code",
			IsError: true,
		}, nil
	}

	// Build command: claude --print "prompt"
	cmdArgs := []string{"--print", params.Prompt}

	if params.MaxTokens > 0 {
		cmdArgs = append(cmdArgs, "--max-tokens", fmt.Sprintf("%d", params.MaxTokens))
	}

	// Create command with timeout.
	timeout := 5 * time.Minute
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, claudePath, cmdArgs...)
	if params.WorkingDir != "" {
		cmd.Dir = params.WorkingDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Check if it was a timeout.
		if cmdCtx.Err() == context.DeadlineExceeded {
			return &tools.Result{
				Content: fmt.Sprintf("claude command timed out after %s", timeout),
				IsError: true,
			}, nil
		}

		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &tools.Result{
			Content: fmt.Sprintf("claude command failed: %s", strings.TrimSpace(errMsg)),
			IsError: true,
		}, nil
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = "(claude returned empty output)"
	}

	// Truncate if too long.
	const maxOutput = 50000
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n\n... (output truncated)"
	}

	return &tools.Result{Content: output}, nil
}

// RegisterClaudeCode registers the claude_code tool with the given registry.
func RegisterClaudeCode(registry *tools.Registry) {
	registry.Register(&ClaudeCodeTool{})
}
