package environments

import "strings"

// ReasoningEnv provides a reasoning and problem-solving evaluation environment.
// It rewards step-by-step thinking, logical deduction, and correct conclusions.
type ReasoningEnv struct {
	BaseEnv
	ProblemType  string // "math", "logic", "coding", "general"
	Difficulty   int    // 1-5
	RequireSteps bool
}

// NewReasoningEnv creates a reasoning evaluation environment.
func NewReasoningEnv(problemType string) *ReasoningEnv {
	base := NewBaseEnv()
	base.Name = "reasoning"
	base.Temperature = 0.3 // Lower temperature for reasoning tasks
	return &ReasoningEnv{
		BaseEnv:      *base,
		ProblemType:  problemType,
		Difficulty:   3,
		RequireSteps: true,
	}
}

// ReasoningReward evaluates reasoning quality.
func ReasoningReward(state State, action Action) float64 {
	reward := 0.0

	if action.Type != "message" {
		return reward
	}

	content := action.Content
	lower := strings.ToLower(content)

	// Reward for step-by-step reasoning
	stepIndicators := []string{"step 1", "first,", "next,", "then,", "finally,",
		"therefore", "because", "since", "given that", "let's think"}
	stepsFound := 0
	for _, indicator := range stepIndicators {
		if strings.Contains(lower, indicator) {
			stepsFound++
		}
	}
	if stepsFound >= 2 {
		reward += 0.3
	}

	// Reward for structured output
	if strings.Contains(content, "\n") {
		lineCount := strings.Count(content, "\n")
		if lineCount >= 3 && lineCount <= 30 {
			reward += 0.2
		}
	}

	// Reward for showing work / intermediate results
	workIndicators := []string{"=", "→", "->", "result:", "answer:", "solution:"}
	for _, indicator := range workIndicators {
		if strings.Contains(lower, indicator) {
			reward += 0.05
		}
	}

	// Reward for appropriate length (reasoning should be thorough)
	if len(content) > 200 && len(content) < 5000 {
		reward += 0.2
	}

	// Penalty for vague responses
	vagueIndicators := []string{"i think maybe", "it might be", "possibly",
		"not sure but", "hard to say"}
	for _, indicator := range vagueIndicators {
		if strings.Contains(lower, indicator) {
			reward -= 0.1
		}
	}

	// Cap reward
	if reward > 1.0 {
		reward = 1.0
	}

	return reward
}

// NewReasoningEnvironment creates the environment with its reward function.
func NewReasoningEnvironment() *Environment {
	return NewEnvironment("reasoning", "Reasoning and problem-solving evaluation", ReasoningReward)
}
