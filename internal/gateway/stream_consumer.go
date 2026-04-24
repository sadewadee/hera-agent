package gateway

import (
	"context"
	"github.com/sadewadee/hera/internal/llm"
)

// StreamConsumer consumes LLM streaming events and forwards them to a platform adapter.
type StreamConsumer struct {
	gateway  *Gateway
	platform string
	chatID   string
}

// NewStreamConsumer creates a consumer that forwards stream events to the given platform/chat.
func NewStreamConsumer(gw *Gateway, platform, chatID string) *StreamConsumer {
	return &StreamConsumer{gateway: gw, platform: platform, chatID: chatID}
}

// Consume reads from the stream channel and sends accumulated text to the platform.
func (sc *StreamConsumer) Consume(ctx context.Context, ch <-chan llm.StreamEvent) error {
	var fullText string
	for ev := range ch {
		if ev.Type == "delta" { fullText += ev.Delta }
		if ev.Type == "done" || ev.Type == "error" { break }
	}
	if fullText == "" { return nil }
	return sc.gateway.SendTo(ctx, sc.platform, sc.chatID, OutgoingMessage{Text: fullText})
}
