package agent

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/skills"
	"github.com/sadewadee/hera/internal/tools"
)

// --- Test doubles ---

type mockLLMProvider struct {
	chatFn   func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error)
	tokensFn func(messages []llm.Message) (int, error)
}

func (m *mockLLMProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.chatFn != nil {
		return m.chatFn(ctx, req)
	}
	return &llm.ChatResponse{
		Message: llm.Message{
			Role:    llm.RoleAssistant,
			Content: "Mock response",
		},
		FinishReason: "stop",
	}, nil
}

func (m *mockLLMProvider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	ch := make(chan llm.StreamEvent, 1)
	ch <- llm.StreamEvent{Type: "delta", Delta: "Mock stream"}
	close(ch)
	return ch, nil
}

func (m *mockLLMProvider) CountTokens(messages []llm.Message) (int, error) {
	if m.tokensFn != nil {
		return m.tokensFn(messages)
	}
	return 10, nil
}

func (m *mockLLMProvider) ModelInfo() llm.ModelMetadata {
	return llm.ModelMetadata{
		ID:            "mock-model",
		Provider:      "mock",
		ContextWindow: 4096,
		MaxOutput:     1024,
		SupportsTools: true,
	}
}

type mockMemoryProvider struct{}

func (m *mockMemoryProvider) SaveFact(ctx context.Context, userID, key, value string) error {
	return nil
}
func (m *mockMemoryProvider) GetFacts(ctx context.Context, userID string) ([]memory.Fact, error) {
	return nil, nil
}
func (m *mockMemoryProvider) Search(ctx context.Context, query string, opts memory.SearchOpts) ([]memory.MemoryResult, error) {
	return nil, nil
}
func (m *mockMemoryProvider) SaveConversation(ctx context.Context, sessionID string, messages []llm.Message) error {
	return nil
}
func (m *mockMemoryProvider) GetConversation(ctx context.Context, sessionID string) ([]llm.Message, error) {
	return nil, nil
}
func (m *mockMemoryProvider) SessionSearch(ctx context.Context, query string) ([]memory.SessionResult, error) {
	return nil, nil
}
func (m *mockMemoryProvider) SaveNote(ctx context.Context, note memory.Note) error { return nil }
func (m *mockMemoryProvider) UpdateNote(ctx context.Context, userID, name, description, content string) error {
	return nil
}
func (m *mockMemoryProvider) DeleteNote(ctx context.Context, userID, name string) error { return nil }
func (m *mockMemoryProvider) GetNote(ctx context.Context, userID, name string) (*memory.Note, error) {
	return nil, nil
}
func (m *mockMemoryProvider) ListNotes(ctx context.Context, userID string, typ memory.NoteType) ([]memory.Note, error) {
	return nil, nil
}
func (m *mockMemoryProvider) ListUserSessions(ctx context.Context, userID string, limit int) ([]memory.SessionSummary, error) {
	return nil, nil
}
func (m *mockMemoryProvider) Close() error { return nil }

type mockMemorySummarizer struct{}

func (m *mockMemorySummarizer) Summarize(ctx context.Context, messages []llm.Message) (string, error) {
	return "summary", nil
}

// --- Tests ---

func newTestAgent(t *testing.T, llmProvider llm.Provider) *Agent {
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

func TestAgent_HandleMessage(t *testing.T) {
	t.Run("returns assistant response for simple message", func(t *testing.T) {
		llmP := &mockLLMProvider{
			chatFn: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
				return &llm.ChatResponse{
					Message: llm.Message{
						Role:    llm.RoleAssistant,
						Content: "Hello! How can I help?",
					},
					FinishReason: "stop",
				}, nil
			},
		}
		agent := newTestAgent(t, llmP)

		resp, err := agent.HandleMessage(context.Background(), "telegram", "chat1", "user1", "Hi")
		if err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if resp != "Hello! How can I help?" {
			t.Errorf("response = %q, want %q", resp, "Hello! How can I help?")
		}
	})

	t.Run("executes tool calls and returns final response", func(t *testing.T) {
		// callCount is read/written by the Chat mock from both the main
		// HandleMessage goroutine and the background title-generation
		// goroutine that HandleMessage spawns after the first turn.
		// Atomic makes that race-free.
		var callCount atomic.Int64
		llmP := &mockLLMProvider{
			chatFn: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
				n := callCount.Add(1)
				if n == 1 {
					// First call: request a tool call
					return &llm.ChatResponse{
						Message: llm.Message{
							Role: llm.RoleAssistant,
							ToolCalls: []llm.ToolCall{
								{
									ID:   "call-1",
									Name: "test_tool",
									Args: json.RawMessage(`{"input": "test"}`),
								},
							},
						},
						FinishReason: "tool_calls",
					}, nil
				}
				// Second call: return final response after tool result
				return &llm.ChatResponse{
					Message: llm.Message{
						Role:    llm.RoleAssistant,
						Content: "Tool result was: tool output",
					},
					FinishReason: "stop",
				}, nil
			},
		}

		agent := newTestAgent(t, llmP)

		// Register a test tool
		agent.tools.Register(&testTool{})

		resp, err := agent.HandleMessage(context.Background(), "telegram", "chat1", "user1", "Use the tool")
		if err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if resp != "Tool result was: tool output" {
			t.Errorf("response = %q, want %q", resp, "Tool result was: tool output")
		}
		// The title-generation goroutine may fire a 3rd Chat call after
		// HandleMessage returns, so allow >= 2 rather than == 2.
		if got := callCount.Load(); got < 2 {
			t.Errorf("LLM called %d times, want at least 2", got)
		}
	})

	t.Run("limits max tool call iterations", func(t *testing.T) {
		llmP := &mockLLMProvider{
			chatFn: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
				// Always return a tool call to trigger the loop limit
				return &llm.ChatResponse{
					Message: llm.Message{
						Role: llm.RoleAssistant,
						ToolCalls: []llm.ToolCall{
							{
								ID:   "call-loop",
								Name: "test_tool",
								Args: json.RawMessage(`{}`),
							},
						},
					},
					FinishReason: "tool_calls",
				}, nil
			},
		}

		agent := newTestAgent(t, llmP)
		agent.tools.Register(&testTool{})

		resp, err := agent.HandleMessage(context.Background(), "telegram", "chat1", "user1", "Loop forever")
		if err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		// Should return something (the last tool call response or a warning)
		_ = resp // just verify no infinite loop
	})

	t.Run("creates session on first message", func(t *testing.T) {
		llmP := &mockLLMProvider{}
		agent := newTestAgent(t, llmP)

		_, err := agent.HandleMessage(context.Background(), "telegram", "chat1", "user1", "Hello")
		if err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}

		// Verify session was created
		sessions := agent.sessions.List()
		if len(sessions) == 0 {
			t.Error("expected at least one session to be created")
		}
	})
}

func TestAgent_HandleStream(t *testing.T) {
	t.Run("streams response events", func(t *testing.T) {
		llmP := &mockLLMProvider{}
		agent := newTestAgent(t, llmP)

		ch, err := agent.HandleStream(context.Background(), "cli", "chat1", "user1", "Hello")
		if err != nil {
			t.Fatalf("HandleStream: %v", err)
		}

		var events []llm.StreamEvent
		for ev := range ch {
			events = append(events, ev)
		}
		if len(events) == 0 {
			t.Error("expected at least one stream event")
		}
	})
}

func TestAgent_NudgeFiring(t *testing.T) {
	t.Run("memory nudge fires at interval", func(t *testing.T) {
		llmP := &mockLLMProvider{
			chatFn: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
				return &llm.ChatResponse{
					Message: llm.Message{
						Role:    llm.RoleAssistant,
						Content: "Response",
					},
					FinishReason: "stop",
				}, nil
			},
		}
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
		cfg.Agent.MemoryNudgeInterval = 2 // Nudge every 2 turns

		agent, err := NewAgent(AgentDeps{
			LLM:      llmP,
			Tools:    toolReg,
			Memory:   memManager,
			Skills:   skillLoader,
			Sessions: sessionMgr,
			Config:   cfg,
		})
		if err != nil {
			t.Fatalf("NewAgent: %v", err)
		}

		// First message (turn 1) -- should NOT contain nudge.
		resp1, err := agent.HandleMessage(context.Background(), "test", "chat1", "nudge-user", "Hello")
		if err != nil {
			t.Fatalf("HandleMessage 1: %v", err)
		}
		if strings.Contains(resp1, "Tip:") {
			t.Error("first message should not contain nudge")
		}

		// Second message (turn 2) -- turn count is now 2, which is divisible by 2, so nudge should fire.
		resp2, err := agent.HandleMessage(context.Background(), "test", "chat1", "nudge-user", "How are you?")
		if err != nil {
			t.Fatalf("HandleMessage 2: %v", err)
		}
		if !strings.Contains(resp2, "Tip:") {
			t.Error("second message should contain memory nudge (MemoryNudgeInterval=2, TurnCount=2)")
		}
		if !strings.Contains(resp2, "remember") {
			t.Error("nudge should mention 'remember'")
		}
	})
}

// TestAgent_MemoryPersistsAcrossRestart verifies that facts and typed notes
// stored in SQLite survive a full agent teardown and recreation. This is the
// regression test for the bug where every restart lost prior memory context.
func TestAgent_MemoryPersistsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := t.TempDir() + "/test.db"
	userID := "persist-test-user"

	// --- Phase 1: First agent instance — seed facts and notes. ---
	sqlite1, err := memory.NewSQLiteProvider(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteProvider: %v", err)
	}
	mgr1 := memory.NewManager(sqlite1, &mockMemorySummarizer{})

	if err := mgr1.SaveFact(ctx, userID, "name", "Alice"); err != nil {
		t.Fatalf("SaveFact: %v", err)
	}
	if err := mgr1.SaveNote(ctx, memory.Note{
		UserID:      userID,
		Type:        memory.NoteTypeUser,
		Name:        "user-profile",
		Description: "senior Go engineer, works on hera",
		Content:     "prefers concise answers",
	}); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}
	if err := mgr1.SaveNote(ctx, memory.Note{
		UserID:      userID,
		Type:        memory.NoteTypeFeedback,
		Name:        "communication-style",
		Description: "always reply in English",
		Content:     "",
	}); err != nil {
		t.Fatalf("SaveNote feedback: %v", err)
	}

	// Tear down first agent instance (simulates process exit / restart).
	if err := sqlite1.Close(); err != nil {
		t.Fatalf("Close sqlite1: %v", err)
	}

	// --- Phase 2: Second agent instance — same DB path, fresh in-memory state. ---
	sqlite2, err := memory.NewSQLiteProvider(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteProvider (restart): %v", err)
	}
	t.Cleanup(func() { sqlite2.Close() })
	mgr2 := memory.NewManager(sqlite2, &mockMemorySummarizer{})

	// Capture the system prompt from the FIRST LLM call (the main chat call,
	// not the background title-generation goroutine that fires afterwards).
	var (
		capturedSystemPrompt string
		callIdx              int32 // atomic counter
	)
	llmP := &mockLLMProvider{
		chatFn: func(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			idx := atomic.AddInt32(&callIdx, 1)
			if idx == 1 {
				// Only capture the first call — that is the real HandleMessage call.
				for _, msg := range req.Messages {
					if msg.Role == llm.RoleSystem {
						capturedSystemPrompt = msg.Content
						break
					}
				}
			}
			return &llm.ChatResponse{
				Message:      llm.Message{Role: llm.RoleAssistant, Content: "ok"},
				FinishReason: "stop",
			}, nil
		},
	}

	a, err := NewAgent(AgentDeps{
		LLM:      llmP,
		Tools:    tools.NewRegistry(),
		Memory:   mgr2,
		Skills:   skills.NewLoader(),
		Sessions: NewSessionManager(30 * time.Minute),
		Config: func() *config.Config {
			c := &config.Config{}
			c.Agent.MaxToolCalls = 1
			return c
		}(),
	})
	if err != nil {
		t.Fatalf("NewAgent restart: %v", err)
	}

	_, err = a.HandleMessage(ctx, "test", "chat1", userID, "hi")
	if err != nil {
		t.Fatalf("HandleMessage after restart: %v", err)
	}

	// The system prompt must contain the persisted fact and both notes.
	checks := []struct {
		label   string
		contain string
	}{
		{"fact name:Alice", "Alice"},
		{"note user-profile", "user-profile"},
		{"note communication-style", "communication-style"},
	}
	for _, c := range checks {
		if !strings.Contains(capturedSystemPrompt, c.contain) {
			t.Errorf("system prompt missing %s: prompt = %q", c.label, capturedSystemPrompt)
		}
	}
}

// testTool is a simple tool for testing.
type testTool struct{}

func (t *testTool) Name() string        { return "test_tool" }
func (t *testTool) Description() string { return "A test tool" }
func (t *testTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type": "object", "properties": {"input": {"type": "string"}}}`)
}
func (t *testTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	return &tools.Result{Content: "tool output"}, nil
}
