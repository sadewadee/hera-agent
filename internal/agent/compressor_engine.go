package agent

import (
	"context"
	"fmt"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/plugins"
)

// CompressorEngine wraps the existing Compressor summarization logic as a
// plugins.ContextEngine. It is the default "compressor" engine registered
// by internal/context.RegisterBuiltinEngines.
type CompressorEngine struct {
	plugins.BaseContextEngine
	summarizer Summarizer
}

var _ plugins.ContextEngine = (*CompressorEngine)(nil)

func NewCompressorEngine(summarizer Summarizer) *CompressorEngine {
	return &CompressorEngine{summarizer: summarizer}
}

func (e *CompressorEngine) Name() string      { return "compressor" }
func (e *CompressorEngine) IsAvailable() bool { return e.summarizer != nil }

func (e *CompressorEngine) Initialize(cfg plugins.ContextEngineConfig) error {
	e.InitBase(cfg)
	return nil
}

func (e *CompressorEngine) UpdateFromResponse(usage llm.Usage) {
	e.RecordUsage(usage)
}

func (e *CompressorEngine) ShouldCompress(promptTokens int) bool {
	return promptTokens > e.ThresholdTokens()
}

// Compress summarizes older messages while preserving the first N (system
// prompt context) and last N (recent conversation). Both N values are counted
// in individual messages, not turn-pairs.
func (e *CompressorEngine) Compress(ctx context.Context, messages []llm.Message, _ int) ([]llm.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	protectFirst := e.ProtectFirstN()
	protectLast := e.ProtectLastN()

	if protectFirst+protectLast >= len(messages) {
		return messages, nil
	}

	head := messages[:protectFirst]
	toSummarize := messages[protectFirst : len(messages)-protectLast]
	tail := messages[len(messages)-protectLast:]

	summary, err := e.summarizer.Summarize(ctx, toSummarize)
	if err != nil {
		return nil, fmt.Errorf("compressor engine: summarize: %w", err)
	}

	e.IncrCompressionCount()

	result := make([]llm.Message, 0, len(head)+1+len(tail))
	result = append(result, head...)
	result = append(result, llm.Message{
		Role:    llm.RoleSystem,
		Content: fmt.Sprintf("[Conversation Summary]\n%s", summary),
	})
	result = append(result, tail...)

	return result, nil
}
