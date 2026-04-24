package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/internal/tools"
)

// mockProvider implements memory.Provider for testing.
type mockProvider struct {
	facts          map[string]map[string]string // userID -> key -> value
	searchResults  []memory.MemoryResult
	searchErr      error
	saveFactErr    error
	conversations  map[string][]llm.Message
	sessionResults []memory.SessionResult
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		facts:         make(map[string]map[string]string),
		conversations: make(map[string][]llm.Message),
	}
}

func (m *mockProvider) SaveFact(_ context.Context, userID, key, value string) error {
	if m.saveFactErr != nil {
		return m.saveFactErr
	}
	if m.facts[userID] == nil {
		m.facts[userID] = make(map[string]string)
	}
	m.facts[userID][key] = value
	return nil
}

func (m *mockProvider) GetFacts(_ context.Context, userID string) ([]memory.Fact, error) {
	var facts []memory.Fact
	if m.facts[userID] != nil {
		for k, v := range m.facts[userID] {
			facts = append(facts, memory.Fact{UserID: userID, Key: k, Value: v})
		}
	}
	return facts, nil
}

func (m *mockProvider) Search(_ context.Context, query string, opts memory.SearchOpts) ([]memory.MemoryResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchResults, nil
}

func (m *mockProvider) SaveConversation(_ context.Context, sessionID string, messages []llm.Message) error {
	m.conversations[sessionID] = messages
	return nil
}

func (m *mockProvider) GetConversation(_ context.Context, sessionID string) ([]llm.Message, error) {
	return m.conversations[sessionID], nil
}

func (m *mockProvider) SessionSearch(_ context.Context, query string) ([]memory.SessionResult, error) {
	return m.sessionResults, nil
}

func (m *mockProvider) SaveNote(_ context.Context, note memory.Note) error { return nil }
func (m *mockProvider) UpdateNote(_ context.Context, userID, name, description, content string) error {
	return nil
}
func (m *mockProvider) DeleteNote(_ context.Context, userID, name string) error { return nil }
func (m *mockProvider) GetNote(_ context.Context, userID, name string) (*memory.Note, error) {
	return nil, nil
}
func (m *mockProvider) ListNotes(_ context.Context, userID string, typ memory.NoteType) ([]memory.Note, error) {
	return nil, nil
}
func (m *mockProvider) ListUserSessions(_ context.Context, userID string, limit int) ([]memory.SessionSummary, error) {
	return nil, nil
}

func (m *mockProvider) Close() error { return nil }

// mockSummarizer implements memory.Summarizer for testing.
type mockSummarizer struct{}

func (s *mockSummarizer) Summarize(_ context.Context, messages []llm.Message) (string, error) {
	return "mock summary", nil
}

// --- MemorySaveTool tests ---

func TestMemorySaveTool_Name(t *testing.T) {
	tool := &MemorySaveTool{}
	if got := tool.Name(); got != "memory_save" {
		t.Errorf("Name() = %q, want %q", got, "memory_save")
	}
}

func TestMemorySaveTool_Description(t *testing.T) {
	tool := &MemorySaveTool{}
	if got := tool.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

func TestMemorySaveTool_Parameters(t *testing.T) {
	tool := &MemorySaveTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("Parameters() returned invalid JSON")
	}
}

func TestMemorySaveTool_Execute_SavesFact(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySaveTool{manager: mgr}

	args, _ := json.Marshal(memorySaveArgs{Key: "name", Value: "Alice", UserID: "user1"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "remembered") {
		t.Errorf("result should confirm save, got: %q", result.Content)
	}
	if !strings.Contains(result.Content, "name") || !strings.Contains(result.Content, "Alice") {
		t.Errorf("result should contain key and value, got: %q", result.Content)
	}

	// Verify the fact was saved
	if provider.facts["user1"]["name"] != "Alice" {
		t.Errorf("fact not saved correctly, got: %v", provider.facts)
	}
}

func TestMemorySaveTool_Execute_DefaultUserID(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySaveTool{manager: mgr}

	args, _ := json.Marshal(memorySaveArgs{Key: "lang", Value: "Go"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}

	// Should use "default" as user ID
	if provider.facts["default"]["lang"] != "Go" {
		t.Errorf("expected fact saved under 'default' user, got: %v", provider.facts)
	}
}

func TestMemorySaveTool_Execute_MissingKey(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySaveTool{manager: mgr}

	args, _ := json.Marshal(memorySaveArgs{Key: "", Value: "Alice"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for missing key")
	}
}

func TestMemorySaveTool_Execute_MissingValue(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySaveTool{manager: mgr}

	args, _ := json.Marshal(memorySaveArgs{Key: "name", Value: ""})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for missing value")
	}
}

func TestMemorySaveTool_Execute_SaveError(t *testing.T) {
	provider := newMockProvider()
	provider.saveFactErr = fmt.Errorf("database error")
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySaveTool{manager: mgr}

	args, _ := json.Marshal(memorySaveArgs{Key: "name", Value: "Alice"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error when save fails")
	}
	if !strings.Contains(result.Content, "save fact") {
		t.Errorf("error should mention save fact, got: %q", result.Content)
	}
}

func TestMemorySaveTool_Execute_InvalidJSON(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySaveTool{manager: mgr}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for invalid JSON")
	}
}

// --- MemorySearchTool tests ---

func TestMemorySearchTool_Name(t *testing.T) {
	tool := &MemorySearchTool{}
	if got := tool.Name(); got != "memory_search" {
		t.Errorf("Name() = %q, want %q", got, "memory_search")
	}
}

func TestMemorySearchTool_Description(t *testing.T) {
	tool := &MemorySearchTool{}
	if got := tool.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

func TestMemorySearchTool_Parameters(t *testing.T) {
	tool := &MemorySearchTool{}
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("Parameters() returned invalid JSON")
	}
}

func TestMemorySearchTool_Execute_ReturnsResults(t *testing.T) {
	provider := newMockProvider()
	provider.searchResults = []memory.MemoryResult{
		{Content: "User prefers Go", Source: "fact", Score: 0.95},
		{Content: "Discussed testing", Source: "conversation", Score: 0.80},
	}
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySearchTool{manager: mgr}

	args, _ := json.Marshal(memorySearchArgs{Query: "Go programming"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Found 2 memories") {
		t.Errorf("result should mention 2 memories, got: %q", result.Content)
	}
	if !strings.Contains(result.Content, "User prefers Go") {
		t.Errorf("result should contain memory content, got: %q", result.Content)
	}
	if !strings.Contains(result.Content, "0.95") {
		t.Errorf("result should contain score, got: %q", result.Content)
	}
}

func TestMemorySearchTool_Execute_NoResults(t *testing.T) {
	provider := newMockProvider()
	provider.searchResults = []memory.MemoryResult{}
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySearchTool{manager: mgr}

	args, _ := json.Marshal(memorySearchArgs{Query: "unknown topic"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "no memories found") {
		t.Errorf("result should say no memories found, got: %q", result.Content)
	}
}

func TestMemorySearchTool_Execute_EmptyQuery(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySearchTool{manager: mgr}

	args, _ := json.Marshal(memorySearchArgs{Query: ""})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for empty query")
	}
}

func TestMemorySearchTool_Execute_SearchError(t *testing.T) {
	provider := newMockProvider()
	provider.searchErr = fmt.Errorf("search failed")
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySearchTool{manager: mgr}

	args, _ := json.Marshal(memorySearchArgs{Query: "test"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error when search fails")
	}
}

func TestMemorySearchTool_Execute_DefaultLimit(t *testing.T) {
	provider := newMockProvider()
	provider.searchResults = []memory.MemoryResult{
		{Content: "result 1", Source: "fact", Score: 0.9},
	}
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySearchTool{manager: mgr}

	// No limit specified -- should default to 5
	args, _ := json.Marshal(memorySearchArgs{Query: "test"})
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("Execute() returned error: %s", result.Content)
	}
}

func TestMemorySearchTool_Execute_InvalidJSON(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	tool := &MemorySearchTool{manager: mgr}

	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.IsError {
		t.Error("Execute() should return error for invalid JSON")
	}
}

// --- RegisterMemory tests ---

func TestRegisterMemory(t *testing.T) {
	provider := newMockProvider()
	mgr := memory.NewManager(provider, &mockSummarizer{})
	registry := tools.NewRegistry()
	RegisterMemory(registry, mgr)

	if _, ok := registry.Get("memory_save"); !ok {
		t.Error("memory_save tool not registered")
	}
	if _, ok := registry.Get("memory_search"); !ok {
		t.Error("memory_search tool not registered")
	}
}
