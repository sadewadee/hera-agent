package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sadewadee/hera/internal/llm"
)

// mockSummarizer is a test double that returns a fixed summary.
type mockSummarizer struct {
	summary string
	err     error
	called  bool
	msgs    []llm.Message
}

func (m *mockSummarizer) Summarize(ctx context.Context, messages []llm.Message) (string, error) {
	m.called = true
	m.msgs = messages
	return m.summary, m.err
}

func TestCompressor_Compress(t *testing.T) {
	t.Run("returns messages unchanged when under threshold", func(t *testing.T) {
		summarizer := &mockSummarizer{summary: "summary"}
		c := NewCompressor(summarizer, 100000, 3) // huge threshold

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Hello", Timestamp: time.Now()},
			{Role: llm.RoleAssistant, Content: "Hi", Timestamp: time.Now()},
		}

		result, err := c.Compress(context.Background(), messages)
		if err != nil {
			t.Fatalf("Compress: %v", err)
		}
		if len(result) != len(messages) {
			t.Errorf("expected %d messages, got %d", len(messages), len(result))
		}
		if summarizer.called {
			t.Error("summarizer should not be called when under threshold")
		}
	})

	t.Run("compresses when over threshold", func(t *testing.T) {
		summarizer := &mockSummarizer{summary: "Previous conversation summary."}
		c := NewCompressor(summarizer, 20, 2) // very low threshold, protect last 2 turns

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "This is a long message that should push us over the token threshold for compression", Timestamp: time.Now()},
			{Role: llm.RoleAssistant, Content: "This is another long response that adds many tokens to the context", Timestamp: time.Now()},
			{Role: llm.RoleUser, Content: "Third message", Timestamp: time.Now()},
			{Role: llm.RoleAssistant, Content: "Fourth message", Timestamp: time.Now()},
			{Role: llm.RoleUser, Content: "Recent question", Timestamp: time.Now()},
			{Role: llm.RoleAssistant, Content: "Recent answer", Timestamp: time.Now()},
		}

		result, err := c.Compress(context.Background(), messages)
		if err != nil {
			t.Fatalf("Compress: %v", err)
		}
		if !summarizer.called {
			t.Fatal("expected summarizer to be called")
		}
		// The result should start with a summary message and keep the last protected turns
		if len(result) == 0 {
			t.Fatal("expected non-empty result")
		}
		if result[0].Role != llm.RoleSystem {
			t.Errorf("first message role = %q, want %q", result[0].Role, llm.RoleSystem)
		}
		if !strings.Contains(result[0].Content, "Previous conversation summary.") {
			t.Error("first message should contain the summary")
		}
		// Last messages should be the protected recent turns
		last := result[len(result)-1]
		if last.Content != "Recent answer" {
			t.Errorf("last message = %q, want %q", last.Content, "Recent answer")
		}
	})

	t.Run("protects last N turns", func(t *testing.T) {
		summarizer := &mockSummarizer{summary: "Summary"}
		c := NewCompressor(summarizer, 10, 1) // protect 1 turn = 2 messages (user+assistant)

		messages := []llm.Message{
			{Role: llm.RoleUser, Content: "Old message with enough content to exceed threshold", Timestamp: time.Now()},
			{Role: llm.RoleAssistant, Content: "Old reply with enough content to exceed threshold", Timestamp: time.Now()},
			{Role: llm.RoleUser, Content: "Latest", Timestamp: time.Now()},
			{Role: llm.RoleAssistant, Content: "Reply", Timestamp: time.Now()},
		}

		result, err := c.Compress(context.Background(), messages)
		if err != nil {
			t.Fatalf("Compress: %v", err)
		}
		// Should have summary + the last 2 messages (1 turn)
		if len(result) < 3 {
			t.Fatalf("expected at least 3 messages (summary + 2 protected), got %d", len(result))
		}
		// Last two should be the protected turn
		if result[len(result)-2].Content != "Latest" {
			t.Errorf("expected protected user message 'Latest', got %q", result[len(result)-2].Content)
		}
		if result[len(result)-1].Content != "Reply" {
			t.Errorf("expected protected assistant message 'Reply', got %q", result[len(result)-1].Content)
		}
	})

	t.Run("handles empty messages", func(t *testing.T) {
		summarizer := &mockSummarizer{summary: "Summary"}
		c := NewCompressor(summarizer, 100, 2)

		result, err := c.Compress(context.Background(), nil)
		if err != nil {
			t.Fatalf("Compress: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
	})
}

func TestEstimateTokensForMessages(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "abcdefgh"},      // 8 chars = 2 tokens
		{Role: llm.RoleAssistant, Content: "abcdefgh"}, // 8 chars = 2 tokens
	}

	got := EstimateTokensForMessages(messages)
	if got != 4 {
		t.Errorf("EstimateTokensForMessages = %d, want 4", got)
	}
}
