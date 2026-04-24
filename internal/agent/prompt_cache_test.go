package agent

import (
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPromptCaching_Disabled(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system prompt"},
		{Role: llm.RoleUser, Content: "hello"},
	}
	result := ApplyPromptCaching(msgs, PromptCacheConfig{Enabled: false})
	assert.Equal(t, msgs, result)
}

func TestApplyPromptCaching_EmptyMessages(t *testing.T) {
	result := ApplyPromptCaching(nil, PromptCacheConfig{Enabled: true})
	assert.Nil(t, result)

	result = ApplyPromptCaching([]llm.Message{}, PromptCacheConfig{Enabled: true})
	assert.Empty(t, result)
}

func TestApplyPromptCaching_DefaultBreakpoint(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system"},
		{Role: llm.RoleUser, Content: "user1"},
		{Role: llm.RoleAssistant, Content: "asst1"},
		{Role: llm.RoleUser, Content: "user2"},
		{Role: llm.RoleAssistant, Content: "asst2"},
	}
	result := ApplyPromptCaching(msgs, PromptCacheConfig{Enabled: true})
	require.Len(t, result, 5)
	// First 3 messages should have cache hint annotation.
	for i := 0; i < 3; i++ {
		assert.Equal(t, "__cache_hint__", result[i].Name, "message %d should be annotated", i)
	}
	// Messages after breakpoint should be untouched.
	assert.NotEqual(t, "__cache_hint__", result[3].Name)
	assert.NotEqual(t, "__cache_hint__", result[4].Name)
}

func TestApplyPromptCaching_CustomBreakpoint(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system"},
		{Role: llm.RoleUser, Content: "user1"},
		{Role: llm.RoleAssistant, Content: "asst1"},
		{Role: llm.RoleUser, Content: "user2"},
	}
	cfg := PromptCacheConfig{Enabled: true, StaticBreakpointIndex: 2}
	result := ApplyPromptCaching(msgs, cfg)
	require.Len(t, result, 4)
	assert.Equal(t, "__cache_hint__", result[0].Name)
	assert.Equal(t, "__cache_hint__", result[1].Name)
	assert.NotEqual(t, "__cache_hint__", result[2].Name)
}

func TestApplyPromptCaching_BreakpointExceedsLength(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system"},
	}
	cfg := PromptCacheConfig{Enabled: true, StaticBreakpointIndex: 100}
	result := ApplyPromptCaching(msgs, cfg)
	require.Len(t, result, 1)
	assert.Equal(t, "__cache_hint__", result[0].Name)
}

func TestApplyPromptCaching_DoesNotMutateOriginal(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system", Name: "original"},
		{Role: llm.RoleUser, Content: "user1"},
	}
	_ = ApplyPromptCaching(msgs, PromptCacheConfig{Enabled: true, StaticBreakpointIndex: 2})
	// Original should be untouched.
	assert.Equal(t, "original", msgs[0].Name)
}

func TestApplyPromptCaching_SkipsToolMessages(t *testing.T) {
	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system"},
		{Role: llm.RoleTool, Content: "tool result", ToolCallID: "call_1"},
		{Role: llm.RoleUser, Content: "user1"},
	}
	result := ApplyPromptCaching(msgs, PromptCacheConfig{Enabled: true})
	// Tool message should NOT get cache_hint because it has ToolCallID or Role==Tool.
	assert.Equal(t, "__cache_hint__", result[0].Name)
	assert.NotEqual(t, "__cache_hint__", result[1].Name)
}

func TestEstimateCacheSavings_Empty(t *testing.T) {
	cached, total, ratio := EstimateCacheSavings(nil, 3)
	assert.Equal(t, 0, cached)
	assert.Equal(t, 0, total)
	assert.Equal(t, 0.0, ratio)
}

func TestEstimateCacheSavings_Default(t *testing.T) {
	msgs := []llm.Message{
		{Content: "aaaa"},             // 4 chars -> 1 token
		{Content: "bbbbbbbb"},         // 8 chars -> 2 tokens
		{Content: "cccccccccccc"},     // 12 chars -> 3 tokens
		{Content: "dddddddddddddddd"}, // 16 chars -> 4 tokens
	}
	cached, total, ratio := EstimateCacheSavings(msgs, 0) // uses default breakpoint=3
	// Cached: msgs 0,1,2 = 1+2+3 = 6 tokens
	// Total: 1+2+3+4 = 10 tokens
	assert.Equal(t, 6, cached)
	assert.Equal(t, 10, total)
	assert.InDelta(t, 0.6, ratio, 0.01)
}

func TestEstimateCacheSavings_CustomBreakpoint(t *testing.T) {
	msgs := []llm.Message{
		{Content: "aaaa"},
		{Content: "bbbb"},
		{Content: "cccc"},
		{Content: "dddd"},
	}
	cached, total, ratio := EstimateCacheSavings(msgs, 2)
	// Each msg: 4 chars / 4 = 1 token
	assert.Equal(t, 2, cached)
	assert.Equal(t, 4, total)
	assert.InDelta(t, 0.5, ratio, 0.01)
}

func TestEstimateCacheSavings_BreakpointExceedsLength(t *testing.T) {
	msgs := []llm.Message{
		{Content: "aaaa"},
	}
	cached, total, ratio := EstimateCacheSavings(msgs, 100)
	assert.Equal(t, 1, cached)
	assert.Equal(t, 1, total)
	assert.InDelta(t, 1.0, ratio, 0.01)
}

func TestEstimateCacheSavings_EmptyContent(t *testing.T) {
	msgs := []llm.Message{
		{Content: ""},
		{Content: ""},
	}
	// Empty content gets minimum 1 token each.
	cached, total, _ := EstimateCacheSavings(msgs, 1)
	assert.Equal(t, 1, cached)
	assert.Equal(t, 2, total)
}
