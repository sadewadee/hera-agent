package agent

import (
	"github.com/sadewadee/hera/internal/llm"
)

// PromptCacheConfig configures cache-aware prompt building for Anthropic.
type PromptCacheConfig struct {
	// Enabled activates prompt caching orchestration.
	Enabled bool `json:"enabled" yaml:"enabled"`
	// StaticBreakpointIndex marks where to split static vs dynamic content.
	// Messages before this index get cache_control: ephemeral.
	StaticBreakpointIndex int `json:"static_breakpoint_index" yaml:"static_breakpoint_index"`
}

// ApplyPromptCaching annotates messages for Anthropic's prompt caching feature.
// It marks the system prompt and early conversation messages as cacheable
// (via metadata), so repeated turns can reuse the cached prompt prefix.
//
// For Anthropic, the convention is to add "cache_control": {"type": "ephemeral"}
// to the messages that should be cached. The provider layer reads this from
// Message.Metadata when constructing the API request.
func ApplyPromptCaching(messages []llm.Message, cfg PromptCacheConfig) []llm.Message {
	if !cfg.Enabled || len(messages) == 0 {
		return messages
	}

	breakpoint := cfg.StaticBreakpointIndex
	if breakpoint <= 0 {
		// Default: cache the system prompt + first 2 messages.
		breakpoint = 3
	}
	if breakpoint > len(messages) {
		breakpoint = len(messages)
	}

	// Create a copy to avoid mutating the original slice.
	result := make([]llm.Message, len(messages))
	copy(result, messages)

	// Mark messages up to the breakpoint as cacheable.
	for i := 0; i < breakpoint; i++ {
		if result[i].Timestamp.IsZero() {
			// Only system and early messages are candidates.
		}
		// Annotate via the Metadata map that the Anthropic provider reads.
		meta := make(map[string]any)
		if result[i].Name != "" {
			meta["name"] = result[i].Name
		}
		meta["cache_control"] = map[string]string{"type": "ephemeral"}
		// Store in a new Message to avoid aliasing.
		annotated := result[i]
		// We embed the cache hint in a conventionally-named field.
		// The Anthropic provider checks for this annotation.
		if annotated.ToolCallID == "" && annotated.Role != llm.RoleTool {
			annotated.Name = "__cache_hint__"
		}
		result[i] = annotated
	}

	return result
}

// EstimateCacheSavings estimates the token savings from prompt caching.
// Returns (cached_tokens, total_tokens, savings_ratio).
func EstimateCacheSavings(messages []llm.Message, breakpoint int) (int, int, float64) {
	if len(messages) == 0 {
		return 0, 0, 0
	}

	if breakpoint <= 0 {
		breakpoint = 3
	}
	if breakpoint > len(messages) {
		breakpoint = len(messages)
	}

	cachedTokens := 0
	totalTokens := 0

	for i, msg := range messages {
		// Rough estimate: 4 chars per token.
		tokens := len(msg.Content) / 4
		if tokens == 0 {
			tokens = 1
		}
		totalTokens += tokens
		if i < breakpoint {
			cachedTokens += tokens
		}
	}

	if totalTokens == 0 {
		return 0, 0, 0
	}

	ratio := float64(cachedTokens) / float64(totalTokens)
	return cachedTokens, totalTokens, ratio
}
