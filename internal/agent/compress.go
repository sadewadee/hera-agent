package agent

import (
	"context"
	"fmt"

	"github.com/sadewadee/hera/internal/llm"
)

// Summarizer is the interface for generating conversation summaries.
// This mirrors memory.Summarizer but is declared here to avoid circular imports.
type Summarizer interface {
	Summarize(ctx context.Context, messages []llm.Message) (string, error)
}

// Compressor manages context window compression by summarizing old messages.
type Compressor struct {
	summarizer     Summarizer
	tokenThreshold int // max tokens before compression triggers
	protectedTurns int // number of recent turns to protect from compression
}

// NewCompressor creates a new compressor.
// tokenThreshold: compress when total tokens exceed this.
// protectedTurns: number of recent user/assistant turn pairs to keep intact.
func NewCompressor(summarizer Summarizer, tokenThreshold int, protectedTurns int) *Compressor {
	if protectedTurns < 1 {
		protectedTurns = 1
	}
	return &Compressor{
		summarizer:     summarizer,
		tokenThreshold: tokenThreshold,
		protectedTurns: protectedTurns,
	}
}

// Compress checks if messages exceed the token threshold and compresses if needed.
// It protects the last N turns (user+assistant pairs) and summarizes the rest.
func (c *Compressor) Compress(ctx context.Context, messages []llm.Message) ([]llm.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	totalTokens := EstimateTokensForMessages(messages)
	if totalTokens <= c.tokenThreshold {
		return messages, nil
	}

	// Calculate how many messages to protect (last N turns = N * 2 messages for user+assistant).
	protectedCount := c.protectedTurns * 2
	if protectedCount >= len(messages) {
		// Not enough messages to compress; return as-is.
		return messages, nil
	}

	// Split into messages to summarize and messages to keep.
	toSummarize := messages[:len(messages)-protectedCount]
	toKeep := messages[len(messages)-protectedCount:]

	// Generate summary of the older messages.
	summary, err := c.summarizer.Summarize(ctx, toSummarize)
	if err != nil {
		return nil, fmt.Errorf("summarize messages: %w", err)
	}

	// Build result: summary as system message + protected messages.
	summaryMsg := llm.Message{
		Role:    llm.RoleSystem,
		Content: fmt.Sprintf("[Conversation Summary]\n%s", summary),
	}

	result := make([]llm.Message, 0, 1+len(toKeep))
	result = append(result, summaryMsg)
	result = append(result, toKeep...)

	return result, nil
}
