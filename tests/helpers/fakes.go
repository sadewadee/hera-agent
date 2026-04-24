package testhelpers

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/sadewadee/hera/internal/gateway"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
)

// --- FakeLLMProvider ---

// FakeLLMProvider implements llm.Provider for testing without real API calls.
type FakeLLMProvider struct {
	mu            sync.Mutex
	ChatFn        func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error)
	ChatStreamFn  func(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error)
	CountTokensFn func(messages []llm.Message) (int, error)
	ModelInfoFn   func() llm.ModelMetadata

	// ChatCalls records each ChatRequest for inspection.
	ChatCalls []llm.ChatRequest
}

// NewFakeLLMProvider creates a FakeLLMProvider that returns a static response.
func NewFakeLLMProvider(response string) *FakeLLMProvider {
	return &FakeLLMProvider{
		ChatFn: func(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			return &llm.ChatResponse{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: response,
				},
				Usage: llm.Usage{
					PromptTokens:     10,
					CompletionTokens: 5,
					TotalTokens:      15,
				},
				Model:        "test-model",
				FinishReason: "stop",
			}, nil
		},
	}
}

// NewFakeLLMProviderWithToolCall creates a provider that returns a tool call
// on the first request, then a text response on the second.
func NewFakeLLMProviderWithToolCall(toolName, toolArgs, finalResponse string) *FakeLLMProvider {
	callCount := 0
	return &FakeLLMProvider{
		ChatFn: func(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &llm.ChatResponse{
					Message: llm.Message{
						Role: llm.RoleAssistant,
						ToolCalls: []llm.ToolCall{
							{
								ID:   "tc_001",
								Name: toolName,
								Args: json.RawMessage(toolArgs),
							},
						},
					},
					Usage: llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
				}, nil
			}
			return &llm.ChatResponse{
				Message: llm.Message{
					Role:    llm.RoleAssistant,
					Content: finalResponse,
				},
				Usage: llm.Usage{PromptTokens: 15, CompletionTokens: 10, TotalTokens: 25},
			}, nil
		},
	}
}

func (p *FakeLLMProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	p.mu.Lock()
	p.ChatCalls = append(p.ChatCalls, req)
	fn := p.ChatFn
	p.mu.Unlock()
	if fn != nil {
		return fn(ctx, req)
	}
	return &llm.ChatResponse{
		Message: llm.Message{Role: llm.RoleAssistant, Content: "default response"},
		Usage:   llm.Usage{PromptTokens: 5, CompletionTokens: 5, TotalTokens: 10},
	}, nil
}

func (p *FakeLLMProvider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	if p.ChatStreamFn != nil {
		return p.ChatStreamFn(ctx, req)
	}
	ch := make(chan llm.StreamEvent, 3)
	go func() {
		defer close(ch)
		ch <- llm.StreamEvent{Type: "delta", Delta: "streaming "}
		ch <- llm.StreamEvent{Type: "delta", Delta: "response"}
		ch <- llm.StreamEvent{Type: "done", Usage: &llm.Usage{PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8}}
	}()
	return ch, nil
}

func (p *FakeLLMProvider) CountTokens(messages []llm.Message) (int, error) {
	if p.CountTokensFn != nil {
		return p.CountTokensFn(messages)
	}
	total := 0
	for _, m := range messages {
		total += 4 + len(m.Content)/4
	}
	if total == 0 && len(messages) > 0 {
		total = 1
	}
	return total, nil
}

func (p *FakeLLMProvider) ModelInfo() llm.ModelMetadata {
	if p.ModelInfoFn != nil {
		return p.ModelInfoFn()
	}
	return llm.ModelMetadata{
		ID:            "test-model",
		Provider:      "test",
		ContextWindow: 8192,
		MaxOutput:     2048,
		SupportsTools: true,
	}
}

// --- FakeAdapter ---

// FakeAdapter implements gateway.PlatformAdapter for testing.
type FakeAdapter struct {
	mu          sync.Mutex
	name        string
	connected   bool
	handler     gateway.MessageHandler
	SentMsgs    []SentMessage
	ConnectErr  error
	SendErr     error
	ChatInfoMap map[string]*gateway.ChatInfo
}

// SentMessage records a message sent via the adapter.
type SentMessage struct {
	ChatID  string
	Message gateway.OutgoingMessage
}

// NewFakeAdapter creates a fake adapter with the given name.
func NewFakeAdapter(name string) *FakeAdapter {
	return &FakeAdapter{
		name:        name,
		ChatInfoMap: make(map[string]*gateway.ChatInfo),
	}
}

func (a *FakeAdapter) Name() string { return a.name }

func (a *FakeAdapter) Connect(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ConnectErr != nil {
		return a.ConnectErr
	}
	a.connected = true
	return nil
}

func (a *FakeAdapter) Disconnect(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.connected = false
	return nil
}

func (a *FakeAdapter) IsConnected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.connected
}

func (a *FakeAdapter) Send(_ context.Context, chatID string, msg gateway.OutgoingMessage) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.SendErr != nil {
		return a.SendErr
	}
	a.SentMsgs = append(a.SentMsgs, SentMessage{ChatID: chatID, Message: msg})
	return nil
}

func (a *FakeAdapter) OnMessage(handler gateway.MessageHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handler = handler
}

func (a *FakeAdapter) GetChatInfo(_ context.Context, chatID string) (*gateway.ChatInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	info, ok := a.ChatInfoMap[chatID]
	if !ok {
		return &gateway.ChatInfo{ID: chatID, Platform: a.name, Type: "private"}, nil
	}
	return info, nil
}

func (a *FakeAdapter) SupportedMedia() []gateway.MediaType {
	return []gateway.MediaType{gateway.MediaPhoto, gateway.MediaFile}
}

// SimulateIncoming sends a simulated incoming message through the adapter's handler.
func (a *FakeAdapter) SimulateIncoming(msg gateway.IncomingMessage) {
	a.mu.Lock()
	h := a.handler
	a.mu.Unlock()
	if h != nil {
		h(context.Background(), msg)
	}
}

// GetSentMessages returns a copy of all messages sent through this adapter.
func (a *FakeAdapter) GetSentMessages() []SentMessage {
	a.mu.Lock()
	defer a.mu.Unlock()
	cp := make([]SentMessage, len(a.SentMsgs))
	copy(cp, a.SentMsgs)
	return cp
}

// --- FakeMemoryProvider ---

// FakeMemoryProvider implements memory.Provider in-memory for testing.
type FakeMemoryProvider struct {
	mu            sync.Mutex
	Facts         map[string][]memory.Fact // userID -> facts
	Conversations map[string][]llm.Message // sessionID -> messages
	SearchResults []memory.MemoryResult
	SessionHits   []memory.SessionResult
	SavedFacts    []savedFact
}

type savedFact struct {
	UserID string
	Key    string
	Value  string
}

// NewFakeMemoryProvider creates an in-memory fake provider.
func NewFakeMemoryProvider() *FakeMemoryProvider {
	return &FakeMemoryProvider{
		Facts:         make(map[string][]memory.Fact),
		Conversations: make(map[string][]llm.Message),
	}
}

func (p *FakeMemoryProvider) SaveFact(_ context.Context, userID, key, value string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Facts[userID] = append(p.Facts[userID], memory.Fact{
		UserID:    userID,
		Key:       key,
		Value:     value,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	p.SavedFacts = append(p.SavedFacts, savedFact{UserID: userID, Key: key, Value: value})
	return nil
}

func (p *FakeMemoryProvider) GetFacts(_ context.Context, userID string) ([]memory.Fact, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Facts[userID], nil
}

func (p *FakeMemoryProvider) Search(_ context.Context, _ string, _ memory.SearchOpts) ([]memory.MemoryResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.SearchResults, nil
}

func (p *FakeMemoryProvider) SaveConversation(_ context.Context, sessionID string, messages []llm.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make([]llm.Message, len(messages))
	copy(cp, messages)
	p.Conversations[sessionID] = cp
	return nil
}

func (p *FakeMemoryProvider) GetConversation(_ context.Context, sessionID string) ([]llm.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Conversations[sessionID], nil
}

func (p *FakeMemoryProvider) SessionSearch(_ context.Context, _ string) ([]memory.SessionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.SessionHits, nil
}

func (p *FakeMemoryProvider) SaveNote(_ context.Context, _ memory.Note) error { return nil }
func (p *FakeMemoryProvider) UpdateNote(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (p *FakeMemoryProvider) DeleteNote(_ context.Context, _, _ string) error { return nil }
func (p *FakeMemoryProvider) GetNote(_ context.Context, _, _ string) (*memory.Note, error) {
	return nil, nil
}
func (p *FakeMemoryProvider) ListNotes(_ context.Context, _ string, _ memory.NoteType) ([]memory.Note, error) {
	return nil, nil
}
func (p *FakeMemoryProvider) ListUserSessions(_ context.Context, _ string, _ int) ([]memory.SessionSummary, error) {
	return nil, nil
}

func (p *FakeMemoryProvider) Close() error { return nil }

// --- FakeSummarizer ---

// FakeSummarizer implements memory.Summarizer for testing.
type FakeSummarizer struct {
	Result string
	Err    error
	Calls  int
}

func (s *FakeSummarizer) Summarize(_ context.Context, _ []llm.Message) (string, error) {
	s.Calls++
	return s.Result, s.Err
}
