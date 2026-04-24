package environments

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// SingularityEnvironment executes commands via Singularity containers.
type SingularityEnvironment struct {
	ImagePath string
	BindPaths []string
}

func (e *SingularityEnvironment) Name() string { return "singularity" }

func (e *SingularityEnvironment) Execute(ctx context.Context, command string, args []string) (*ExecResult, error) {
	singArgs := []string{"exec"}
	for _, bp := range e.BindPaths {
		singArgs = append(singArgs, "--bind", bp)
	}
	singArgs = append(singArgs, e.ImagePath, command)
	singArgs = append(singArgs, args...)
	cmd := exec.CommandContext(ctx, "singularity", singArgs...)
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

func (e *SingularityEnvironment) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := e.Execute(ctx, "cat", []string{path})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read %s: %s", path, res.Stderr)
	}
	return []byte(res.Stdout), nil
}

func (e *SingularityEnvironment) WriteFile(ctx context.Context, path string, data []byte) error {
	singArgs := []string{"exec", e.ImagePath, "tee", path}
	cmd := exec.CommandContext(ctx, "singularity", singArgs...)
	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}

func (e *SingularityEnvironment) Cleanup(_ context.Context) error { return nil }
