package environments

// BaseEnv provides the base environment configuration for RL environments.
type BaseEnv struct {
	Name         string
	SystemPrompt string
	Tools        []string
	MaxTurns     int
	Temperature  float64
}

// NewBaseEnv creates a base environment with default settings.
func NewBaseEnv() *BaseEnv {
	return &BaseEnv{
		Name:        "base",
		MaxTurns:    50,
		Temperature: 0.7,
		Tools:       []string{"file_read", "file_write", "run_command", "web_search"},
	}
}

// GetConfig returns the environment configuration as a map.
func (e *BaseEnv) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"name": e.Name, "system_prompt": e.SystemPrompt,
		"tools": e.Tools, "max_turns": e.MaxTurns, "temperature": e.Temperature,
	}
}
