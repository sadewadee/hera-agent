package llm

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider is a test implementation of Provider.
type mockProvider struct {
	modelID string
}

func (m *mockProvider) Chat(_ context.Context, req ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{
		Message: Message{Role: RoleAssistant, Content: "mock response"},
		Model:   m.modelID,
	}, nil
}

func (m *mockProvider) ChatStream(_ context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: "done"}
	close(ch)
	return ch, nil
}

func (m *mockProvider) CountTokens(messages []Message) (int, error) {
	return len(messages) * 10, nil
}

func (m *mockProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{ID: m.modelID, Provider: "mock"}
}

func TestRegistry_RegisterAndCreate(t *testing.T) {
	r := NewRegistry()

	r.Register("mock", func(cfg ProviderConfig) (Provider, error) {
		return &mockProvider{modelID: cfg.Model}, nil
	})

	p, err := r.Create("mock", ProviderConfig{Model: "gpt-4o"})
	require.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "gpt-4o", p.ModelInfo().ID)
}

func TestRegistry_CreateUnknownProvider(t *testing.T) {
	r := NewRegistry()
	_, err := r.Create("nonexistent", ProviderConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider")
}

func TestRegistry_CreateFactoryError(t *testing.T) {
	r := NewRegistry()
	r.Register("failing", func(cfg ProviderConfig) (Provider, error) {
		return nil, fmt.Errorf("factory failed")
	})

	_, err := r.Create("failing", ProviderConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "factory failed")
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register("mock", func(cfg ProviderConfig) (Provider, error) {
		return &mockProvider{modelID: "test-model"}, nil
	})

	_, err := r.Create("mock", ProviderConfig{Model: "test-model"})
	require.NoError(t, err)

	p, ok := r.Get("mock")
	assert.True(t, ok)
	assert.NotNil(t, p)
}

func TestRegistry_GetNotFound(t *testing.T) {
	r := NewRegistry()
	p, ok := r.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, p)
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register("provider-a", func(cfg ProviderConfig) (Provider, error) {
		return &mockProvider{}, nil
	})
	r.Register("provider-b", func(cfg ProviderConfig) (Provider, error) {
		return &mockProvider{}, nil
	})

	names := r.List()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "provider-a")
	assert.Contains(t, names, "provider-b")
}

func TestRegistry_ListEmpty(t *testing.T) {
	r := NewRegistry()
	names := r.List()
	assert.Empty(t, names)
}

func TestProviderConfig(t *testing.T) {
	cfg := ProviderConfig{
		APIKey:  "sk-test",
		BaseURL: "https://api.example.com",
		Model:   "gpt-4o",
		OrgID:   "org-123",
		Timeout: 30,
	}
	assert.Equal(t, "sk-test", cfg.APIKey)
	assert.Equal(t, "gpt-4o", cfg.Model)
	assert.Equal(t, 30, cfg.Timeout)
}

func TestMockProvider_Chat(t *testing.T) {
	p := &mockProvider{modelID: "test"}
	resp, err := p.Chat(context.Background(), ChatRequest{
		Model:    "test",
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	require.NoError(t, err)
	assert.Equal(t, RoleAssistant, resp.Message.Role)
}

func TestMockProvider_ChatStream(t *testing.T) {
	p := &mockProvider{modelID: "test"}
	ch, err := p.ChatStream(context.Background(), ChatRequest{})
	require.NoError(t, err)

	var events []StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}
	assert.NotEmpty(t, events)
}

func TestMockProvider_CountTokens(t *testing.T) {
	p := &mockProvider{modelID: "test"}
	count, err := p.CountTokens([]Message{
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, Content: "world"},
	})
	require.NoError(t, err)
	assert.Equal(t, 20, count)
}
