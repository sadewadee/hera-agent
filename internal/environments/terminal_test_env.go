package environments

// TerminalTestEnv provides an environment for testing terminal-based tools.
type TerminalTestEnv struct {
	BaseEnv
	AllowedCommands []string
	Sandbox         bool
}

// NewTerminalTestEnv creates a terminal test environment.
func NewTerminalTestEnv() *TerminalTestEnv {
	base := NewBaseEnv()
	base.Name = "terminal-test"
	base.Tools = []string{"run_command", "terminal", "file_read"}
	return &TerminalTestEnv{BaseEnv: *base, Sandbox: true}
}
