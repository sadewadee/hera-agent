package environments

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DefaultCompressionConfig ---

func TestDefaultCompressionConfig(t *testing.T) {
	cfg := DefaultCompressionConfig()
	assert.Equal(t, 15250, cfg.TargetMaxTokens)
	assert.Equal(t, 750, cfg.SummaryTargetTokens)
	assert.True(t, cfg.ProtectFirstSystem)
	assert.True(t, cfg.ProtectFirstHuman)
	assert.True(t, cfg.ProtectFirstGPT)
	assert.True(t, cfg.ProtectFirstTool)
	assert.Equal(t, 4, cfg.ProtectLastNTurns)
	assert.True(t, cfg.AddSummaryNotice)
	assert.True(t, cfg.SkipUnderTarget)
	assert.Equal(t, "_compressed", cfg.OutputSuffix)
}

// --- SimpleTokenCounter ---

func TestSimpleTokenCounter_Words(t *testing.T) {
	tc := &SimpleTokenCounter{}
	tokens := tc.CountTokens("hello world how are you")
	// 5 words * 1.3 = 6.5 -> 6
	assert.Equal(t, 6, tokens)
}

func TestSimpleTokenCounter_Empty(t *testing.T) {
	tc := &SimpleTokenCounter{}
	tokens := tc.CountTokens("")
	assert.Equal(t, 0, tokens)
}

func TestSimpleTokenCounter_SingleChar(t *testing.T) {
	tc := &SimpleTokenCounter{}
	tokens := tc.CountTokens("a")
	// 1 word * 1.3 = 1.3 -> 1
	assert.Equal(t, 1, tokens)
}

// --- AggregateMetrics ---

func TestAggregateMetrics_AddTrajectory(t *testing.T) {
	agg := &AggregateMetrics{}
	agg.AddTrajectory(TrajectoryMetrics{
		OriginalTokens:   1000,
		CompressedTokens: 500,
		TokensSaved:      500,
		OriginalTurns:    20,
		CompressedTurns:  10,
		TurnsRemoved:     10,
		WasCompressed:    true,
	})
	agg.AddTrajectory(TrajectoryMetrics{
		OriginalTokens:     500,
		CompressedTokens:   500,
		SkippedUnderTarget: true,
	})

	assert.Equal(t, 2, agg.TotalTrajectories)
	assert.Equal(t, 1, agg.TrajectoriesCompressed)
	assert.Equal(t, 1, agg.TrajectoriesSkippedUnder)
	assert.Equal(t, 1500, agg.TotalTokensBefore)
	assert.Equal(t, 1000, agg.TotalTokensAfter)
	assert.Equal(t, 500, agg.TotalTokensSaved)
}

// --- NewTrajectoryCompressor ---

func TestNewTrajectoryCompressor_DefaultTokenizer(t *testing.T) {
	tc := NewTrajectoryCompressor(DefaultCompressionConfig(), nil)
	require.NotNil(t, tc)
	assert.NotNil(t, tc.tokenizer)
}

func TestNewTrajectoryCompressor_CustomTokenizer(t *testing.T) {
	custom := &SimpleTokenCounter{}
	tc := NewTrajectoryCompressor(DefaultCompressionConfig(), custom)
	assert.Equal(t, custom, tc.tokenizer)
}

// --- CompressTrajectory ---

func TestCompressTrajectory_UnderTarget_Skipped(t *testing.T) {
	cfg := DefaultCompressionConfig()
	cfg.TargetMaxTokens = 100000
	tc := NewTrajectoryCompressor(cfg, nil)

	turns := []ConversationTurn{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result, metrics := tc.CompressTrajectory(turns)
	assert.Equal(t, len(turns), len(result))
	assert.True(t, metrics.SkippedUnderTarget)
	assert.False(t, metrics.WasCompressed)
	assert.Equal(t, 1.0, metrics.CompressionRatio)
}

func TestCompressTrajectory_OverTarget_Compressed(t *testing.T) {
	cfg := DefaultCompressionConfig()
	cfg.TargetMaxTokens = 10 // Very small target
	cfg.SummaryTargetTokens = 2
	cfg.ProtectLastNTurns = 1
	tc := NewTrajectoryCompressor(cfg, nil)

	turns := []ConversationTurn{
		{Role: "system", Content: "You are a helpful assistant with expertise."},
		{Role: "user", Content: "Tell me about Go programming language in detail."},
		{Role: "assistant", Content: "Go is a statically typed compiled language designed at Google."},
		{Role: "tool", Content: "Tool result with lots of data that should be compressed away.", ToolName: "web_search"},
		{Role: "assistant", Content: "Based on the search results, here is more info about Go."},
		{Role: "tool", Content: "Another tool result with even more data to compress.", ToolName: "file_read"},
		{Role: "assistant", Content: "Here is the final answer about Go programming."},
	}

	result, metrics := tc.CompressTrajectory(turns)
	assert.True(t, metrics.WasCompressed)
	assert.Less(t, len(result), len(turns))
	assert.Greater(t, metrics.TurnsRemoved, 0)
}

func TestCompressTrajectory_Empty(t *testing.T) {
	tc := NewTrajectoryCompressor(DefaultCompressionConfig(), nil)
	result, metrics := tc.CompressTrajectory(nil)
	assert.Nil(t, result)
	assert.True(t, metrics.SkippedUnderTarget)
}

// --- GetAggregateMetrics ---

func TestGetAggregateMetrics_AccumulatesAcrossCompresses(t *testing.T) {
	cfg := DefaultCompressionConfig()
	cfg.TargetMaxTokens = 5
	cfg.SummaryTargetTokens = 1
	cfg.ProtectLastNTurns = 1
	tc := NewTrajectoryCompressor(cfg, nil)

	// Build turns with enough middle content to trigger real compression.
	turns := makeLargeTurns()

	_, m1 := tc.CompressTrajectory(turns)
	_, m2 := tc.CompressTrajectory(turns)

	agg := tc.GetAggregateMetrics()
	// Aggregate only counts trajectories that WasCompressed=true.
	expectedCount := 0
	if m1.WasCompressed {
		expectedCount++
	}
	if m2.WasCompressed {
		expectedCount++
	}
	assert.Equal(t, expectedCount, agg.TrajectoriesCompressed)
}

func makeLargeTurns() []ConversationTurn {
	turns := []ConversationTurn{
		{Role: "system", Content: "System prompt here."},
		{Role: "user", Content: "User question here."},
	}
	// Add enough middle turns to exceed target.
	for i := 0; i < 20; i++ {
		turns = append(turns, ConversationTurn{
			Role:     "assistant",
			Content:  strings.Repeat("Some long assistant response text. ", 20),
			ToolName: "",
		})
		turns = append(turns, ConversationTurn{
			Role:     "tool",
			Content:  strings.Repeat("Tool result data block. ", 20),
			ToolName: "search",
		})
	}
	turns = append(turns, ConversationTurn{Role: "assistant", Content: "Final answer."})
	return turns
}

// --- identifyHeadProtected ---

func TestIdentifyHeadProtected_AllTypes(t *testing.T) {
	cfg := DefaultCompressionConfig()
	tc := NewTrajectoryCompressor(cfg, nil)

	turns := []ConversationTurn{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "usr"},
		{Role: "assistant", Content: "asst"},
		{Role: "tool", Content: "tool"},
		{Role: "user", Content: "second user"},
	}

	count := tc.identifyHeadProtected(turns)
	assert.Equal(t, 4, count) // system + user + assistant + tool
}

func TestIdentifyHeadProtected_BreaksOnUnprotected(t *testing.T) {
	cfg := DefaultCompressionConfig()
	cfg.ProtectFirstGPT = false
	tc := NewTrajectoryCompressor(cfg, nil)

	turns := []ConversationTurn{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "usr"},
		{Role: "assistant", Content: "asst"}, // Not protected
		{Role: "tool", Content: "tool"},
	}

	count := tc.identifyHeadProtected(turns)
	assert.Equal(t, 2, count) // Breaks at assistant
}

// --- buildSummaryMessage ---

func TestBuildSummaryMessage(t *testing.T) {
	cfg := DefaultCompressionConfig()
	tc := NewTrajectoryCompressor(cfg, nil)

	turns := []ConversationTurn{
		{Role: "assistant", Content: "I will search."},
		{Role: "tool", Content: "result1", ToolName: "web_search"},
		{Role: "tool", Content: "result2", ToolName: "file_read"},
		{Role: "tool", Content: "result3", ToolName: "web_search"}, // duplicate tool
	}

	summary := tc.buildSummaryMessage(turns)
	assert.Equal(t, "user", summary.Role)
	assert.Contains(t, summary.Content, "4 conversation turns compressed")
	assert.Contains(t, summary.Content, "web_search")
	assert.Contains(t, summary.Content, "file_read")
	assert.Contains(t, summary.Content, "summarised")
}
