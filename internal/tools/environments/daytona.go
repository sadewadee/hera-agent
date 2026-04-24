package environments

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// DaytonaEnvironment executes commands via the Daytona CLI.
type DaytonaEnvironment struct {
	WorkspaceID string
}

func (e *DaytonaEnvironment) Name() string { return "daytona" }

func (e *DaytonaEnvironment) Execute(ctx context.Context, command string, args []string) (*ExecResult, error) {
	daytonaArgs := []string{"exec", e.WorkspaceID, "--"}
	daytonaArgs = append(daytonaArgs, command)
	daytonaArgs = append(daytonaArgs, args...)
	cmd := exec.CommandContext(ctx, "daytona", daytonaArgs...)
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

func (e *DaytonaEnvironment) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := e.Execute(ctx, "cat", []string{path})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read %s: %s", path, res.Stderr)
	}
	return []byte(res.Stdout), nil
}

func (e *DaytonaEnvironment) WriteFile(ctx context.Context, path string, data []byte) error {
	daytonaArgs := []string{"exec", e.WorkspaceID, "--", "tee", path}
	cmd := exec.CommandContext(ctx, "daytona", daytonaArgs...)
	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}

func (e *DaytonaEnvironment) Cleanup(_ context.Context) error { return nil }
