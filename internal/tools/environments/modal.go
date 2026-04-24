package environments

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ModalEnvironment executes commands via the Modal CLI.
type ModalEnvironment struct {
	AppName string
	GPU     string
}

func (e *ModalEnvironment) Name() string { return "modal" }

func (e *ModalEnvironment) Execute(ctx context.Context, command string, args []string) (*ExecResult, error) {
	modalArgs := []string{"run"}
	if e.GPU != "" {
		modalArgs = append(modalArgs, "--gpu", e.GPU)
	}
	remoteCmd := command + " " + strings.Join(args, " ")
	modalArgs = append(modalArgs, "--command", remoteCmd)
	cmd := exec.CommandContext(ctx, "modal", modalArgs...)
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

func (e *ModalEnvironment) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := e.Execute(ctx, "cat", []string{path})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read %s: %s", path, res.Stderr)
	}
	return []byte(res.Stdout), nil
}

func (e *ModalEnvironment) WriteFile(ctx context.Context, path string, data []byte) error {
	modalArgs := []string{"run", "--command", fmt.Sprintf("cat > %s", path)}
	cmd := exec.CommandContext(ctx, "modal", modalArgs...)
	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}

func (e *ModalEnvironment) Cleanup(_ context.Context) error { return nil }
