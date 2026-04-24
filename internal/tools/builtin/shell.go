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

// dangerousPatterns are shell patterns that require explicit approval.
var dangerousPatterns = []string{
	"rm -rf",
	"rm -fr",
	"rmdir",
	"mkfs",
	"dd if=",
	"> /dev/",
	":(){ :|:& };:",
	"chmod -R 777",
	"DROP TABLE",
	"DROP DATABASE",
	"TRUNCATE",
	"DELETE FROM",
	"shutdown",
	"reboot",
	"halt",
	"init 0",
	"init 6",
	"kill -9",
	"pkill -9",
	"killall",
	"format c:",
	"fdisk",
	"wget|sh",
	"curl|sh",
	"wget|bash",
	"curl|bash",
}

// pipeToShellPatterns detects commands piped into a shell interpreter.
// These catch patterns like "curl ... | bash" with spaces around the pipe.
var pipeToShellPatterns = []struct {
	source string
	shell  string
}{
	{"wget", "sh"},
	{"wget", "bash"},
	{"curl", "sh"},
	{"curl", "bash"},
}

// RunCommandTool executes shell commands.
type RunCommandTool struct {
	protectedPaths   []string
	dangerousApprove bool
}

type runCommandArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func (r *RunCommandTool) Name() string {
	return "run_command"
}

func (r *RunCommandTool) Description() string {
	return "Executes a shell command and returns its output. Dangerous commands are blocked unless explicitly approved."
}

func (r *RunCommandTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {
				"type": "string",
				"description": "The shell command to execute."
			},
			"timeout": {
				"type": "integer",
				"description": "Timeout in seconds. Defaults to 30."
			}
		},
		"required": ["command"]
	}`)
}

func (r *RunCommandTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var params runCommandArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	if params.Command == "" {
		return &tools.Result{Content: "command is required", IsError: true}, nil
	}

	// Check for dangerous commands
	if pattern, dangerous := isDangerous(params.Command); dangerous && !r.dangerousApprove {
		return &tools.Result{
			Content: fmt.Sprintf("blocked: command matches dangerous pattern %q. Set security.dangerous_approve=true to allow.", pattern),
			IsError: true,
		}, nil
	}

	// Check for protected path access
	if path, protected := accessesProtectedPath(params.Command, r.protectedPaths); protected {
		return &tools.Result{
			Content: fmt.Sprintf("blocked: command accesses protected path %q", path),
			IsError: true,
		}, nil
	}

	timeout := time.Duration(params.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var result strings.Builder
	if stdout.Len() > 0 {
		result.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString("STDERR:\n")
		result.WriteString(stderr.String())
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &tools.Result{
				Content: fmt.Sprintf("command timed out after %v", timeout),
				IsError: true,
			}, nil
		}
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("exit error: %v", err))
		return &tools.Result{Content: result.String(), IsError: true}, nil
	}

	output := result.String()
	if output == "" {
		output = "(no output)"
	}

	// Limit output to avoid blowing up context
	const maxOutput = 50 * 1024
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n... [truncated, output exceeds 50KB]"
	}

	return &tools.Result{Content: output}, nil
}

// isDangerous checks if a command matches any dangerous pattern.
func isDangerous(command string) (string, bool) {
	lower := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return pattern, true
		}
	}
	// Check pipe-to-shell patterns: "curl ... | bash", "wget ... | sh", etc.
	for _, p := range pipeToShellPatterns {
		if strings.Contains(lower, p.source) && strings.Contains(lower, "|") && strings.Contains(lower, p.shell) {
			return p.source + " | " + p.shell, true
		}
	}
	return "", false
}

// accessesProtectedPath checks if a command references a protected path.
func accessesProtectedPath(command string, protectedPaths []string) (string, bool) {
	for _, pp := range protectedPaths {
		expanded, err := expandAndAbs(pp)
		if err != nil {
			continue
		}
		// Check both expanded and unexpanded forms
		if strings.Contains(command, expanded) || strings.Contains(command, pp) {
			return pp, true
		}
	}
	return "", false
}

// RegisterShell registers the run_command tool with the given registry.
func RegisterShell(registry *tools.Registry, protectedPaths []string, dangerousApprove bool) {
	registry.Register(&RunCommandTool{
		protectedPaths:   protectedPaths,
		dangerousApprove: dangerousApprove,
	})
}
