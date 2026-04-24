package environments

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SSHEnvironment executes commands on a remote host via SSH.
type SSHEnvironment struct {
	Host    string
	User    string
	KeyFile string
	Port    string
}

func (e *SSHEnvironment) Name() string { return "ssh" }

func (e *SSHEnvironment) sshArgs() []string {
	args := []string{"-o", "StrictHostKeyChecking=no", "-o", "BatchMode=yes"}
	if e.KeyFile != "" {
		args = append(args, "-i", e.KeyFile)
	}
	if e.Port != "" {
		args = append(args, "-p", e.Port)
	}
	target := e.Host
	if e.User != "" {
		target = e.User + "@" + e.Host
	}
	args = append(args, target)
	return args
}

func (e *SSHEnvironment) Execute(ctx context.Context, command string, args []string) (*ExecResult, error) {
	sshArgs := e.sshArgs()
	remoteCmd := command + " " + strings.Join(args, " ")
	sshArgs = append(sshArgs, remoteCmd)
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
		err = nil
	}
	return &ExecResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: code}, err
}

func (e *SSHEnvironment) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := e.Execute(ctx, "cat", []string{path})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read %s: %s", path, res.Stderr)
	}
	return []byte(res.Stdout), nil
}

func (e *SSHEnvironment) WriteFile(ctx context.Context, path string, data []byte) error {
	sshArgs := e.sshArgs()
	sshArgs = append(sshArgs, fmt.Sprintf("cat > %s", path))
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)
	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}

func (e *SSHEnvironment) Cleanup(_ context.Context) error { return nil }
