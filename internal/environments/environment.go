// Package environments provides reinforcement learning environment types
// and trajectory recording for agent training data generation.
package environments

import (
	"time"
)

// State represents the current environment state for RL evaluation.
type State struct {
	SessionID   string        `json:"session_id"`
	TurnCount   int           `json:"turn_count"`
	Messages    []Message     `json:"messages"`
	ToolsUsed   []string      `json:"tools_used"`
	TokensUsed  int           `json:"tokens_used"`
	ElapsedTime time.Duration `json:"elapsed_time"`
}

// Message is a minimal message representation for environment states.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Action represents an agent action within an environment.
type Action struct {
	Type     string         `json:"type"` // "message", "tool_call", "end_turn"
	Content  string         `json:"content,omitempty"`
	ToolName string         `json:"tool_name,omitempty"`
	ToolArgs map[string]any `json:"tool_args,omitempty"`
}

// RewardFunc evaluates an action taken in a given state and returns a reward.
type RewardFunc func(state State, action Action) float64

// Environment defines a training environment for reinforcement learning.
type Environment struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	RewardFunc  RewardFunc `json:"-"` // not serialized
	MaxSteps    int        `json:"max_steps"`
}

// NewEnvironment creates a new training environment.
func NewEnvironment(name, description string, reward RewardFunc) *Environment {
	return &Environment{
		Name:        name,
		Description: description,
		RewardFunc:  reward,
		MaxSteps:    100,
	}
}

// Evaluate computes the reward for a given state-action pair.
func (e *Environment) Evaluate(state State, action Action) float64 {
	if e.RewardFunc == nil {
		return 0
	}
	return e.RewardFunc(state, action)
}

// DefaultEnvironments returns a set of built-in training environments.
func DefaultEnvironments() []*Environment {
	return []*Environment{
		NewEnvironment(
			"helpfulness",
			"Rewards helpful, relevant, and complete responses",
			func(state State, action Action) float64 {
				if action.Type == "message" && len(action.Content) > 10 {
					return 1.0
				}
				return 0.0
			},
		),
		NewEnvironment(
			"tool_usage",
			"Rewards appropriate and efficient tool usage",
			func(state State, action Action) float64 {
				if action.Type == "tool_call" {
					return 1.0
				}
				return 0.0
			},
		),
		NewEnvironment(
			"efficiency",
			"Rewards concise responses that minimize token usage",
			func(state State, action Action) float64 {
				if action.Type == "message" && len(action.Content) < 500 && len(action.Content) > 10 {
					return 1.0
				}
				return 0.5
			},
		),
	}
}
