package main

import (
	"context"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMMemorySummarizer_Summarize_Error(t *testing.T) {
	t.Parallel()

	// Create summarizer with a mock provider that returns an error.
	s := &llmMemorySummarizer{llm: &mockProvider{err: assert.AnError}}
	_, err := s.Summarize(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "summarize")
}

func TestLLMMemorySummarizer_Summarize_Success(t *testing.T) {
	t.Parallel()

	s := &llmMemorySummarizer{llm: &mockProvider{
		response: llm.ChatResponse{
			Message: llm.Message{Role: llm.RoleAssistant, Content: "summary of conversation"},
		},
	}}

	result, err := s.Summarize(context.Background(), []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
		{Role: llm.RoleAssistant, Content: "Hi there"},
	})
	require.NoError(t, err)
	assert.Equal(t, "summary of conversation", result)
}

// mockProvider implements llm.Provider for testing.
type mockProvider struct {
	response llm.ChatResponse
	err      error
}

func (m *mockProvider) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &m.response, nil
}

func (m *mockProvider) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	return nil, nil
}

func (m *mockProvider) CountTokens(_ []llm.Message) (int, error) { return 0, nil }

func (m *mockProvider) ModelInfo() llm.ModelMetadata {
	return llm.ModelMetadata{ID: "mock", ContextWindow: 4096}
}
