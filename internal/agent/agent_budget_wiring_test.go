package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/tools"
)

// newTestAgentWithBudget creates an Agent with the given BudgetConfig wired in.
func newTestAgentWithBudget(t *testing.T, llmProvider llm.Provider, bc config.BudgetConfig) *Agent {
	t.Helper()
	memProvider := &mockMemoryProvider{}
	memSummarizer := &mockMemorySummarizer{}
	memManager := memory.NewManager(memProvider, memSummarizer)
	toolReg := tools.NewRegistry()
	skillLoader := skills.NewLoader()
	sessionMgr := NewSessionManager(30 * time.Minute)

	cfg := &config.Config{}
	cfg.Agent.MaxToolCalls = 10
	cfg.Agent.Compression.Enabled = false
	cfg.Agent.Personality = "helpful"
	cfg.Agent.Budget = bc

	a, err := NewAgent(AgentDeps{
		LLM:      llmProvider,
		Tools:    toolReg,
		Memory:   memManager,
		Skills:   skillLoader,
		Sessions: sessionMgr,
		Config:   cfg,
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	return a
}

// TestAgent_HandleMessage_BudgetEnforcement verifies that a HandleMessage call
// is rejected with ErrBudgetExceeded when the per-agent token budget has been
// exhausted before the call is made.
func TestAgent_HandleMessage_BudgetEnforcement(t *testing.T) {
	llmP := &mockLLMProvider{
		chatFn: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "should not be returned",
				},
				Usage:        llm.Usage{PromptTokens: 10, CompletionTokens: 5},
				FinishReason: "stop",
			}, nil
		},
	}

	// MaxTokens=1 — the first successful call records 15 tokens, so the
	// second call must be rejected.
	a := newTestAgentWithBudget(t, llmP, config.BudgetConfig{MaxTokens: 1})

	// Pre-load the budget by recording tokens directly, simulating a prior call.
	a.budget.Record(10, 0)

	_, err := a.HandleMessage(context.Background(), "cli", "c1", "u1", "hello")
	if err == nil {
		t.Fatal("expected ErrBudgetExceeded, got nil")
	}
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected errors.Is(err, ErrBudgetExceeded), got: %v", err)
	}
}

// TestAgent_HandleMessage_NoBudgetLimit verifies that when no budget limits are
// configured the agent responds normally (regression guard).
func TestAgent_HandleMessage_NoBudgetLimit(t *testing.T) {
	llmP := &mockLLMProvider{
		chatFn: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "hello back",
				},
				Usage:        llm.Usage{PromptTokens: 5, CompletionTokens: 5},
				FinishReason: "stop",
			}, nil
		},
	}

	// Zero BudgetConfig = unlimited.
	a := newTestAgentWithBudget(t, llmP, config.BudgetConfig{})

	resp, err := a.HandleMessage(context.Background(), "cli", "c1", "u1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "hello back" {
		t.Errorf("resp = %q, want %q", resp, "hello back")
	}
}

// TestAgent_HandleMessage_BudgetRecordsUsage verifies that each HandleMessage
// call records token usage into the budget so Stats() reflects it.
func TestAgent_HandleMessage_BudgetRecordsUsage(t *testing.T) {
	llmP := &mockLLMProvider{
		chatFn: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: "ok",
				},
				Usage:        llm.Usage{PromptTokens: 7, CompletionTokens: 3},
				FinishReason: "stop",
			}, nil
		},
	}

	a := newTestAgentWithBudget(t, llmP, config.BudgetConfig{}) // unlimited

	_, err := a.HandleMessage(context.Background(), "cli", "c1", "u1", "ping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tokens, _ := a.budget.Stats()
	if tokens != 10 {
		t.Errorf("budget.Stats tokens = %d, want 10 (7 prompt + 3 completion)", tokens)
	}
}
