package environments

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// LocalEnvironment executes commands on the local host.
type LocalEnvironment struct {
	WorkDir string
}

func (e *LocalEnvironment) Name() string { return "local" }

func (e *LocalEnvironment) Execute(ctx context.Context, command string, args []string) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if e.WorkDir != "" {
		cmd.Dir = e.WorkDir
	}
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

func (e *LocalEnvironment) ReadFile(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (e *LocalEnvironment) WriteFile(_ context.Context, path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (e *LocalEnvironment) Cleanup(_ context.Context) error { return nil }
