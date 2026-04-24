package environments

// MultiTurnEnv provides a multi-turn conversation evaluation environment.
// It rewards coherent multi-turn dialogues, context retention, and appropriate
// follow-up question handling.
type MultiTurnEnv struct {
	BaseEnv
	MinTurns       int
	MaxTurns       int
	RequireContext bool
}

// NewMultiTurnEnv creates a multi-turn conversation environment.
func NewMultiTurnEnv() *MultiTurnEnv {
	base := NewBaseEnv()
	base.Name = "multi-turn"
	return &MultiTurnEnv{
		BaseEnv:        *base,
		MinTurns:       3,
		MaxTurns:       20,
		RequireContext: true,
	}
}

// MultiTurnReward evaluates multi-turn conversation quality.
// It rewards context retention, coherent follow-ups, and appropriate
// response length across turns.
func MultiTurnReward(state State, action Action) float64 {
	reward := 0.0

	// Reward for maintaining context across turns
	if state.TurnCount > 1 && action.Type == "message" {
		reward += 0.3
	}

	// Reward for appropriate response length (not too short, not too long)
	contentLen := len(action.Content)
	if contentLen > 20 && contentLen < 2000 {
		reward += 0.2
	}

	// Reward for using tools when available
	if action.Type == "tool_call" && action.ToolName != "" {
		reward += 0.2
	}

	// Penalty for ending conversation too early
	if action.Type == "end_turn" && state.TurnCount < 3 {
		reward -= 0.5
	}

	// Penalty for excessive token usage
	if state.TokensUsed > 10000 {
		reward -= 0.1
	}

	return reward
}

// NewMultiTurnEnvironment creates the environment with its reward function.
func NewMultiTurnEnvironment() *Environment {
	return NewEnvironment("multi-turn", "Multi-turn conversation evaluation", MultiTurnReward)
}
