package environments

import "strings"

// CodeReviewEnv provides a code review evaluation environment.
// It rewards identifying bugs, suggesting improvements, and providing
// constructive feedback on code snippets.
type CodeReviewEnv struct {
	BaseEnv
	Language    string
	ReviewDepth string // "surface", "moderate", "deep"
	FocusAreas  []string
}

// NewCodeReviewEnv creates a code review environment.
func NewCodeReviewEnv(language string) *CodeReviewEnv {
	base := NewBaseEnv()
	base.Name = "code-review"
	base.Tools = append(base.Tools, "code_exec", "patch")
	return &CodeReviewEnv{
		BaseEnv:     *base,
		Language:    language,
		ReviewDepth: "moderate",
		FocusAreas:  []string{"bugs", "performance", "readability", "security"},
	}
}

// CodeReviewReward evaluates code review quality.
func CodeReviewReward(state State, action Action) float64 {
	reward := 0.0

	if action.Type != "message" {
		return reward
	}

	content := strings.ToLower(action.Content)

	// Reward for identifying specific issues
	issueIndicators := []string{"bug", "error", "issue", "vulnerability", "race condition",
		"memory leak", "performance", "complexity", "security"}
	for _, indicator := range issueIndicators {
		if strings.Contains(content, indicator) {
			reward += 0.1
		}
	}

	// Reward for suggesting fixes, not just identifying problems
	fixIndicators := []string{"suggest", "recommend", "consider", "instead", "alternative",
		"fix", "improve", "refactor"}
	for _, indicator := range fixIndicators {
		if strings.Contains(content, indicator) {
			reward += 0.1
		}
	}

	// Reward for constructive tone
	constructiveIndicators := []string{"well done", "good approach", "nice", "clean"}
	for _, indicator := range constructiveIndicators {
		if strings.Contains(content, indicator) {
			reward += 0.05
		}
	}

	// Penalty for overly short reviews
	if len(action.Content) < 50 {
		reward -= 0.3
	}

	// Cap reward
	if reward > 1.0 {
		reward = 1.0
	}

	return reward
}

// NewCodeReviewEnvironment creates the environment with its reward function.
func NewCodeReviewEnvironment() *Environment {
	return NewEnvironment("code-review", "Code review evaluation", CodeReviewReward)
}
