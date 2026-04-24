package environments

import "context"

// ExecResult holds the output of a command execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecEnvironment defines the interface for execution backends.
type ExecEnvironment interface {
	// Name returns the environment name.
	Name() string
	// Execute runs a command in the environment.
	Execute(ctx context.Context, command string, args []string) (*ExecResult, error)
	// ReadFile reads a file from the environment.
	ReadFile(ctx context.Context, path string) ([]byte, error)
	// WriteFile writes content to a file in the environment.
	WriteFile(ctx context.Context, path string, data []byte) error
	// Cleanup releases environment resources.
	Cleanup(ctx context.Context) error
}
