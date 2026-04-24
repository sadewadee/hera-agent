package environments

// ToolUseEnv provides a tool use evaluation environment.
// It rewards correct tool selection, proper argument formatting,
// and effective use of tool results in responses.
type ToolUseEnv struct {
	BaseEnv
	RequiredTools []string
	AllowedTools  []string
	MaxToolCalls  int
}

// NewToolUseEnv creates a tool use evaluation environment.
func NewToolUseEnv(allowedTools []string) *ToolUseEnv {
	base := NewBaseEnv()
	base.Name = "tool-use"
	base.Tools = allowedTools
	return &ToolUseEnv{
		BaseEnv:      *base,
		AllowedTools: allowedTools,
		MaxToolCalls: 10,
	}
}

// ToolUseReward evaluates tool usage quality.
func ToolUseReward(state State, action Action) float64 {
	reward := 0.0

	switch action.Type {
	case "tool_call":
		// Reward for using tools
		reward += 0.3

		// Reward for providing arguments
		if len(action.ToolArgs) > 0 {
			reward += 0.2
		}

		// Penalty for repeated tool calls with same arguments
		for _, toolUsed := range state.ToolsUsed {
			if toolUsed == action.ToolName {
				reward -= 0.1
				break
			}
		}

		// Penalty for too many tool calls
		if len(state.ToolsUsed) > 8 {
			reward -= 0.2
		}

	case "message":
		// Reward for synthesizing tool results into a coherent response
		if len(state.ToolsUsed) > 0 && len(action.Content) > 50 {
			reward += 0.3
		}

		// Penalty for responding without using tools when tools were available
		if len(state.ToolsUsed) == 0 && state.TurnCount > 0 {
			reward -= 0.2
		}

	case "end_turn":
		// Reward for ending after productive tool use
		if len(state.ToolsUsed) > 0 {
			reward += 0.1
		}
	}

	return reward
}

// NewToolUseEnvironment creates the environment with its reward function.
func NewToolUseEnvironment() *Environment {
	return NewEnvironment("tool-use", "Tool use evaluation", ToolUseReward)
}
