package environments

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// DockerEnvironment executes commands inside a Docker container.
type DockerEnvironment struct {
	ContainerID string
	Image       string
	WorkDir     string
}

func (e *DockerEnvironment) Name() string { return "docker" }

func (e *DockerEnvironment) Execute(ctx context.Context, command string, args []string) (*ExecResult, error) {
	dockerArgs := []string{"exec"}
	if e.WorkDir != "" {
		dockerArgs = append(dockerArgs, "-w", e.WorkDir)
	}
	dockerArgs = append(dockerArgs, e.ContainerID, command)
	dockerArgs = append(dockerArgs, args...)
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
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

func (e *DockerEnvironment) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := e.Execute(ctx, "cat", []string{path})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("cat %s: %s", path, res.Stderr)
	}
	return []byte(res.Stdout), nil
}

func (e *DockerEnvironment) WriteFile(ctx context.Context, path string, data []byte) error {
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", e.ContainerID, "tee", path)
	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}

func (e *DockerEnvironment) Cleanup(ctx context.Context) error {
	if e.ContainerID != "" {
		return exec.CommandContext(ctx, "docker", "rm", "-f", e.ContainerID).Run()
	}
	return nil
}
