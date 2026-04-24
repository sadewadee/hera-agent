package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sadewadee/hera/internal/tools"
)

// SSHTool executes commands on remote hosts via SSH.
type SSHTool struct{}

type sshToolArgs struct {
	Host         string `json:"host"`
	User         string `json:"user,omitempty"`
	Port         int    `json:"port,omitempty"`
	Command      string `json:"command"`
	IdentityFile string `json:"identity_file,omitempty"`
}

func (t *SSHTool) Name() string { return "ssh" }

func (t *SSHTool) Description() string {
	return "Executes commands on remote hosts via SSH."
}

func (t *SSHTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"host": {
				"type": "string",
				"description": "Remote host address."
			},
			"user": {
				"type": "string",
				"description": "SSH username. Defaults to current user."
			},
			"port": {
				"type": "integer",
				"description": "SSH port. Defaults to 22."
			},
			"command": {
				"type": "string",
				"description": "Command to execute on the remote host."
			},
			"identity_file": {
				"type": "string",
				"description": "Path to SSH private key file."
			}
		},
		"required": ["host", "command"]
	}`)
}

func (t *SSHTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	var a sshToolArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return &tools.Result{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}

	sshArgs := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
	}

	if a.Port > 0 {
		sshArgs = append(sshArgs, "-p", fmt.Sprintf("%d", a.Port))
	}

	if a.IdentityFile != "" {
		sshArgs = append(sshArgs, "-i", a.IdentityFile)
	}

	target := a.Host
	if a.User != "" {
		target = a.User + "@" + a.Host
	}
	sshArgs = append(sshArgs, target, a.Command)

	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		return &tools.Result{Content: fmt.Sprintf("SSH command failed: %v\n%s", err, output), IsError: true}, nil
	}

	// Truncate large output
	if len(output) > 50*1024 {
		output = output[:50*1024] + "\n...[truncated]"
	}

	return &tools.Result{Content: output}, nil
}

// RegisterSSH registers the SSH tool with the given registry.
func RegisterSSH(registry *tools.Registry) {
	registry.Register(&SSHTool{})
}
