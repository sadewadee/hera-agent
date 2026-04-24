package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sadewadee/hera/internal/llm"
)

// BaseContextEngine provides goroutine-safe default implementations for all
// optional ContextEngine methods. Concrete engines embed *BaseContextEngine
// and override only the methods they need.
//
// Required methods NOT implemented here (concrete engines must provide them):
//   - Name() string
//   - IsAvailable() bool
//   - Initialize(cfg ContextEngineConfig) error
//   - UpdateFromResponse(usage llm.Usage)
//   - ShouldCompress(promptTokens int) bool
//   - Compress(ctx, messages, currentTokens) ([]llm.Message, error)
type BaseContextEngine struct {
	mu sync.Mutex

	// State fields — tracked across turns, reset by OnSessionReset.
	lastPromptTokens     int
	lastCompletionTokens int
	lastTotalTokens      int
	thresholdTokens      int
	contextLength        int
	compressionCount     int
	thresholdPercent     float64
	protectFirstN        int
	protectLastN         int

	// configuredName records the name passed to InitBase so Status() can
	// return it even when the concrete engine embeds Base.
	configuredName string
}

// InitBase populates the token-budget state from a ContextEngineConfig.
// Concrete engines should call this from their own Initialize method.
func (b *BaseContextEngine) InitBase(cfg ContextEngineConfig) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.configuredName = cfg.Name
	b.contextLength = cfg.ContextLength

	threshold := cfg.ThresholdPercent
	if threshold <= 0 || threshold > 1 {
		threshold = 0.75
	}
	b.thresholdPercent = threshold

	if cfg.ContextLength > 0 {
		b.thresholdTokens = int(float64(cfg.ContextLength) * threshold)
	}

	b.protectFirstN = cfg.ProtectFirstN
	if b.protectFirstN <= 0 {
		b.protectFirstN = 3
	}
	b.protectLastN = cfg.ProtectLastN
	if b.protectLastN <= 0 {
		b.protectLastN = 6
	}
}

// RecordUsage updates per-turn token counters from a llm.Usage value.
// Concrete engines that embed BaseContextEngine typically call this from
// their own UpdateFromResponse.
func (b *BaseContextEngine) RecordUsage(usage llm.Usage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastPromptTokens = usage.PromptTokens
	b.lastCompletionTokens = usage.CompletionTokens
	b.lastTotalTokens = usage.TotalTokens
}

// IncrCompressionCount increments the compression counter by one.
func (b *BaseContextEngine) IncrCompressionCount() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.compressionCount++
}

// ThresholdTokens returns the configured threshold in absolute tokens.
func (b *BaseContextEngine) ThresholdTokens() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.thresholdTokens
}

// ProtectFirstN returns the configured leading-message protection count.
func (b *BaseContextEngine) ProtectFirstN() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.protectFirstN
}

// ProtectLastN returns the configured trailing-message protection count.
func (b *BaseContextEngine) ProtectLastN() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.protectLastN
}

// --- Default implementations of optional ContextEngine methods ---

// ShouldCompressPreflight returns false by default (no preflight veto).
func (b *BaseContextEngine) ShouldCompressPreflight(_ []llm.Message) bool {
	return false
}

// OnSessionStart is a no-op by default.
func (b *BaseContextEngine) OnSessionStart(_, _, _ string) error {
	return nil
}

// OnSessionEnd is a no-op by default.
func (b *BaseContextEngine) OnSessionEnd(_ string, _ []llm.Message) error {
	return nil
}

// OnSessionReset resets per-session counters to zero.
func (b *BaseContextEngine) OnSessionReset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lastPromptTokens = 0
	b.lastCompletionTokens = 0
	b.lastTotalTokens = 0
	b.compressionCount = 0
}

// GetToolSchemas returns nil by default (engine exposes no tools).
func (b *BaseContextEngine) GetToolSchemas() []llm.ToolDef {
	return nil
}

// HandleToolCall returns an error by default (no tools registered).
func (b *BaseContextEngine) HandleToolCall(_ context.Context, name string, _ json.RawMessage) (string, error) {
	return "", fmt.Errorf("context engine: unknown tool %q", name)
}

// Status returns a snapshot of the engine's current token-budget state.
func (b *BaseContextEngine) Status() ContextEngineStatus {
	b.mu.Lock()
	defer b.mu.Unlock()

	var usagePct float64
	if b.thresholdTokens > 0 {
		usagePct = float64(b.lastPromptTokens) / float64(b.thresholdTokens) * 100
	}

	return ContextEngineStatus{
		Name:                 b.configuredName,
		LastPromptTokens:     b.lastPromptTokens,
		LastCompletionTokens: b.lastCompletionTokens,
		LastTotalTokens:      b.lastTotalTokens,
		ThresholdTokens:      b.thresholdTokens,
		ContextLength:        b.contextLength,
		CompressionCount:     b.compressionCount,
		UsagePercent:         usagePct,
	}
}

// UpdateModel recalculates thresholdTokens from the new context window length.
// Concrete engines that override this should embed and delegate to
// BaseContextEngine.UpdateModel to keep token accounting consistent.
func (b *BaseContextEngine) UpdateModel(_ string, contextLength int, _, _, _ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.contextLength = contextLength
	if contextLength > 0 && b.thresholdPercent > 0 {
		b.thresholdTokens = int(float64(contextLength) * b.thresholdPercent)
	}
	return nil
}

// Shutdown is a no-op by default.
func (b *BaseContextEngine) Shutdown() {}
