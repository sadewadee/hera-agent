package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTitleProvider struct {
	response string
	err      error
}

func (f *fakeTitleProvider) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &llm.ChatResponse{
		Message: llm.Message{Role: llm.RoleAssistant, Content: f.response},
		Usage:   llm.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}, nil
}

func (f *fakeTitleProvider) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	return nil, nil
}
func (f *fakeTitleProvider) CountTokens(_ []llm.Message) (int, error) { return 0, nil }
func (f *fakeTitleProvider) ModelInfo() llm.ModelMetadata {
	return llm.ModelMetadata{ID: "test", Provider: "test"}
}

func TestGenerateTitle_EmptyMessages(t *testing.T) {
	_, err := GenerateTitle(context.Background(), nil, &fakeTitleProvider{response: "Title"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no messages")
}

func TestGenerateTitle_NilProvider(t *testing.T) {
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	_, err := GenerateTitle(context.Background(), msgs, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider is required")
}

func TestGenerateTitle_ReturnsCleanTitle(t *testing.T) {
	provider := &fakeTitleProvider{response: `"Hello World Chat"`}
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "Can you help me with Go testing?"},
		{Role: llm.RoleAssistant, Content: "Sure, I can help with Go testing."},
	}
	title, err := GenerateTitle(context.Background(), msgs, provider)
	require.NoError(t, err)
	assert.Equal(t, "Hello World Chat", title)
}

func TestGenerateTitle_TruncatesLongTitle(t *testing.T) {
	provider := &fakeTitleProvider{response: "This Is A Very Long Title That Has Way Too Many Words In It"}
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	title, err := GenerateTitle(context.Background(), msgs, provider)
	require.NoError(t, err)
	words := strings.Fields(title)
	assert.LessOrEqual(t, len(words), 7)
}

func TestGenerateTitle_EmptyResponseFallback(t *testing.T) {
	provider := &fakeTitleProvider{response: "   "}
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	title, err := GenerateTitle(context.Background(), msgs, provider)
	require.NoError(t, err)
	assert.Equal(t, "New Conversation", title)
}

func TestGenerateTitle_ProviderError(t *testing.T) {
	provider := &fakeTitleProvider{err: assert.AnError}
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	_, err := GenerateTitle(context.Background(), msgs, provider)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "generate title")
}

func TestGenerateTitle_SkipsSystemMessages(t *testing.T) {
	provider := &fakeTitleProvider{response: "Chat About Go"}
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "You are a helpful assistant with a very long system prompt"},
		{Role: llm.RoleUser, Content: "Hello"},
		{Role: llm.RoleAssistant, Content: "Hi"},
	}
	title, err := GenerateTitle(context.Background(), msgs, provider)
	require.NoError(t, err)
	assert.Equal(t, "Chat About Go", title)
}

func TestGenerateTitle_LimitsToSixMessages(t *testing.T) {
	provider := &fakeTitleProvider{response: "Chat Title"}
	msgs := make([]llm.Message, 20)
	for i := range msgs {
		msgs[i] = llm.Message{Role: llm.RoleUser, Content: "message"}
	}
	title, err := GenerateTitle(context.Background(), msgs, provider)
	require.NoError(t, err)
	assert.NotEmpty(t, title)
}

func TestGenerateTitle_StripsBacktickQuotes(t *testing.T) {
	provider := &fakeTitleProvider{response: "`My Title`"}
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hello"}}
	title, err := GenerateTitle(context.Background(), msgs, provider)
	require.NoError(t, err)
	assert.Equal(t, "My Title", title)
}
