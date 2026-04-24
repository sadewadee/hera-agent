package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/sadewadee/hera/internal/llm"
)

// GenerateTitle asks the LLM to produce a short (5 words max) conversation
// title based on the first few messages. This is called after the first
// assistant response in a new session.
func GenerateTitle(ctx context.Context, messages []llm.Message, provider llm.Provider) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages to generate title from")
	}
	if provider == nil {
		return "", fmt.Errorf("LLM provider is required")
	}

	// Take the first few messages for context (up to 6).
	maxMsgs := 6
	if len(messages) < maxMsgs {
		maxMsgs = len(messages)
	}
	sample := messages[:maxMsgs]

	// Build a summary of the conversation start.
	var sb strings.Builder
	for _, m := range sample {
		if m.Role == llm.RoleSystem {
			continue
		}
		content := m.Content
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Role, content))
	}

	titleMessages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: "Generate a short title (5 words maximum) for the following conversation. Return ONLY the title, no quotes, no punctuation, no explanation.",
		},
		{
			Role:    llm.RoleUser,
			Content: sb.String(),
		},
	}

	resp, err := provider.Chat(ctx, llm.ChatRequest{
		Messages:  titleMessages,
		MaxTokens: 32,
	})
	if err != nil {
		return "", fmt.Errorf("generate title: %w", err)
	}

	title := strings.TrimSpace(resp.Message.Content)

	// Clean up the title: remove surrounding quotes if present.
	title = strings.Trim(title, "\"'`")

	// Enforce word limit.
	words := strings.Fields(title)
	if len(words) > 7 {
		words = words[:7]
	}
	title = strings.Join(words, " ")

	if title == "" {
		return "New Conversation", nil
	}

	return title, nil
}
