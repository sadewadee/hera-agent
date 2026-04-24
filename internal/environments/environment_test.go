package environments

import (
	"testing"
	"time"
)

func TestNewEnvironment(t *testing.T) {
	env := NewEnvironment("test", "A test environment", func(s State, a Action) float64 {
		return 1.0
	})

	if env.Name != "test" {
		t.Errorf("Name = %q, want 'test'", env.Name)
	}
	if env.Description != "A test environment" {
		t.Errorf("Description = %q, want 'A test environment'", env.Description)
	}
	if env.MaxSteps != 100 {
		t.Errorf("MaxSteps = %d, want 100", env.MaxSteps)
	}
	if env.RewardFunc == nil {
		t.Error("RewardFunc should not be nil")
	}
}

func TestEnvironment_Evaluate(t *testing.T) {
	env := NewEnvironment("test", "test env", func(s State, a Action) float64 {
		if a.Type == "message" {
			return 2.5
		}
		return 0.0
	})

	state := State{
		SessionID: "s1",
		TurnCount: 1,
	}

	// Test message action.
	reward := env.Evaluate(state, Action{Type: "message", Content: "hello"})
	if reward != 2.5 {
		t.Errorf("Evaluate(message) = %f, want 2.5", reward)
	}

	// Test non-message action.
	reward = env.Evaluate(state, Action{Type: "tool_call"})
	if reward != 0.0 {
		t.Errorf("Evaluate(tool_call) = %f, want 0.0", reward)
	}
}

func TestEnvironment_Evaluate_NilRewardFunc(t *testing.T) {
	env := &Environment{
		Name:        "nil-reward",
		Description: "env with nil reward func",
		RewardFunc:  nil,
	}

	reward := env.Evaluate(State{}, Action{Type: "message"})
	if reward != 0 {
		t.Errorf("Evaluate with nil RewardFunc = %f, want 0", reward)
	}
}

func TestDefaultEnvironments(t *testing.T) {
	envs := DefaultEnvironments()

	if len(envs) != 3 {
		t.Fatalf("DefaultEnvironments() len = %d, want 3", len(envs))
	}

	// Verify names.
	expectedNames := []string{"helpfulness", "tool_usage", "efficiency"}
	for i, env := range envs {
		if env.Name != expectedNames[i] {
			t.Errorf("envs[%d].Name = %q, want %q", i, env.Name, expectedNames[i])
		}
		if env.RewardFunc == nil {
			t.Errorf("envs[%d].RewardFunc should not be nil", i)
		}
	}
}

func TestDefaultEnvironments_Helpfulness(t *testing.T) {
	envs := DefaultEnvironments()
	helpfulness := envs[0]

	state := State{SessionID: "s1"}

	// Long message -> reward 1.0
	reward := helpfulness.Evaluate(state, Action{Type: "message", Content: "This is a helpful response."})
	if reward != 1.0 {
		t.Errorf("helpfulness(long message) = %f, want 1.0", reward)
	}

	// Short message -> reward 0.0
	reward = helpfulness.Evaluate(state, Action{Type: "message", Content: "hi"})
	if reward != 0.0 {
		t.Errorf("helpfulness(short message) = %f, want 0.0", reward)
	}

	// Non-message -> reward 0.0
	reward = helpfulness.Evaluate(state, Action{Type: "tool_call"})
	if reward != 0.0 {
		t.Errorf("helpfulness(tool_call) = %f, want 0.0", reward)
	}
}

func TestDefaultEnvironments_ToolUsage(t *testing.T) {
	envs := DefaultEnvironments()
	toolUsage := envs[1]

	state := State{SessionID: "s1"}

	// Tool call -> reward 1.0
	reward := toolUsage.Evaluate(state, Action{Type: "tool_call", ToolName: "search"})
	if reward != 1.0 {
		t.Errorf("tool_usage(tool_call) = %f, want 1.0", reward)
	}

	// Message -> reward 0.0
	reward = toolUsage.Evaluate(state, Action{Type: "message", Content: "hello"})
	if reward != 0.0 {
		t.Errorf("tool_usage(message) = %f, want 0.0", reward)
	}
}

func TestDefaultEnvironments_Efficiency(t *testing.T) {
	envs := DefaultEnvironments()
	efficiency := envs[2]

	state := State{SessionID: "s1"}

	// Medium message (10-500 chars) -> reward 1.0
	reward := efficiency.Evaluate(state, Action{Type: "message", Content: "A concise but useful response here."})
	if reward != 1.0 {
		t.Errorf("efficiency(concise message) = %f, want 1.0", reward)
	}

	// Too short message -> reward 0.5
	reward = efficiency.Evaluate(state, Action{Type: "message", Content: "hi"})
	if reward != 0.5 {
		t.Errorf("efficiency(short message) = %f, want 0.5", reward)
	}

	// Non-message -> reward 0.5
	reward = efficiency.Evaluate(state, Action{Type: "tool_call"})
	if reward != 0.5 {
		t.Errorf("efficiency(tool_call) = %f, want 0.5", reward)
	}
}

func TestState_Fields(t *testing.T) {
	state := State{
		SessionID:   "session-123",
		TurnCount:   5,
		Messages:    []Message{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}},
		ToolsUsed:   []string{"search", "calculator"},
		TokensUsed:  1500,
		ElapsedTime: 30 * time.Second,
	}

	if state.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want 'session-123'", state.SessionID)
	}
	if state.TurnCount != 5 {
		t.Errorf("TurnCount = %d, want 5", state.TurnCount)
	}
	if len(state.Messages) != 2 {
		t.Errorf("Messages len = %d, want 2", len(state.Messages))
	}
	if len(state.ToolsUsed) != 2 {
		t.Errorf("ToolsUsed len = %d, want 2", len(state.ToolsUsed))
	}
	if state.TokensUsed != 1500 {
		t.Errorf("TokensUsed = %d, want 1500", state.TokensUsed)
	}
}

func TestAction_Fields(t *testing.T) {
	action := Action{
		Type:     "tool_call",
		Content:  "",
		ToolName: "search",
		ToolArgs: map[string]any{"query": "golang testing"},
	}

	if action.Type != "tool_call" {
		t.Errorf("Type = %q, want 'tool_call'", action.Type)
	}
	if action.ToolName != "search" {
		t.Errorf("ToolName = %q, want 'search'", action.ToolName)
	}
	if action.ToolArgs["query"] != "golang testing" {
		t.Errorf("ToolArgs[query] = %v, want 'golang testing'", action.ToolArgs["query"])
	}
}

func TestMessage_Fields(t *testing.T) {
	msg := Message{Role: "user", Content: "hello world"}
	if msg.Role != "user" {
		t.Errorf("Role = %q, want 'user'", msg.Role)
	}
	if msg.Content != "hello world" {
		t.Errorf("Content = %q, want 'hello world'", msg.Content)
	}
}
