package agent

import (
	"context"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSummarizer struct {
	called bool
	result string
}

func (s *stubSummarizer) Summarize(_ context.Context, _ []llm.Message) (string, error) {
	s.called = true
	return s.result, nil
}

func TestCompressorEngine_SatisfiesInterface(t *testing.T) {
	var _ plugins.ContextEngine = (*CompressorEngine)(nil)
}

func TestCompressorEngine_Name(t *testing.T) {
	e := NewCompressorEngine(nil)
	assert.Equal(t, "compressor", e.Name())
}

func TestCompressorEngine_IsAvailable(t *testing.T) {
	assert.False(t, NewCompressorEngine(nil).IsAvailable())
	assert.True(t, NewCompressorEngine(&stubSummarizer{}).IsAvailable())
}

func TestCompressorEngine_InitializeAndThreshold(t *testing.T) {
	e := NewCompressorEngine(&stubSummarizer{})
	err := e.Initialize(plugins.ContextEngineConfig{
		Name:             "compressor",
		ContextLength:    10000,
		ThresholdPercent: 0.80,
		ProtectFirstN:    2,
		ProtectLastN:     4,
	})
	require.NoError(t, err)

	assert.Equal(t, 8000, e.ThresholdTokens())
	assert.Equal(t, 2, e.ProtectFirstN())
	assert.Equal(t, 4, e.ProtectLastN())
}

func TestCompressorEngine_ShouldCompress(t *testing.T) {
	e := NewCompressorEngine(&stubSummarizer{})
	_ = e.Initialize(plugins.ContextEngineConfig{
		ContextLength:    1000,
		ThresholdPercent: 0.75,
	})

	assert.False(t, e.ShouldCompress(700))
	assert.True(t, e.ShouldCompress(800))
}

func TestCompressorEngine_Compress(t *testing.T) {
	sum := &stubSummarizer{result: "older conversation context"}
	e := NewCompressorEngine(sum)
	_ = e.Initialize(plugins.ContextEngineConfig{
		ContextLength:    1000,
		ThresholdPercent: 0.75,
		ProtectFirstN:    1,
		ProtectLastN:     2,
	})

	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system prompt"},
		{Role: llm.RoleUser, Content: "old msg 1"},
		{Role: llm.RoleAssistant, Content: "old resp 1"},
		{Role: llm.RoleUser, Content: "old msg 2"},
		{Role: llm.RoleAssistant, Content: "old resp 2"},
		{Role: llm.RoleUser, Content: "recent msg"},
		{Role: llm.RoleAssistant, Content: "recent resp"},
	}

	result, err := e.Compress(context.Background(), msgs, 0)
	require.NoError(t, err)
	assert.True(t, sum.called)
	// head (1) + summary (1) + tail (2) = 4
	require.Len(t, result, 4)
	assert.Equal(t, "system prompt", result[0].Content)
	assert.Equal(t, llm.RoleSystem, result[1].Role)
	assert.Contains(t, result[1].Content, "Conversation Summary")
	assert.Equal(t, "recent msg", result[2].Content)
	assert.Equal(t, "recent resp", result[3].Content)

	status := e.Status()
	assert.Equal(t, 1, status.CompressionCount)
}

func TestCompressorEngine_CompressProtectsFirstN(t *testing.T) {
	sum := &stubSummarizer{result: "summarized middle"}
	e := NewCompressorEngine(sum)
	_ = e.Initialize(plugins.ContextEngineConfig{
		ContextLength:    1000,
		ThresholdPercent: 0.75,
		ProtectFirstN:    2,
		ProtectLastN:     1,
	})

	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: "system prompt"},
		{Role: llm.RoleUser, Content: "first user msg"},
		{Role: llm.RoleAssistant, Content: "middle resp"},
		{Role: llm.RoleUser, Content: "middle msg"},
		{Role: llm.RoleAssistant, Content: "last resp"},
	}

	result, err := e.Compress(context.Background(), msgs, 0)
	require.NoError(t, err)

	// head (2) + summary (1) + tail (1) = 4
	require.Len(t, result, 4)
	assert.Equal(t, "system prompt", result[0].Content)
	assert.Equal(t, "first user msg", result[1].Content)
	assert.Contains(t, result[2].Content, "Conversation Summary")
	assert.Equal(t, "last resp", result[3].Content)
}

func TestCompressorEngine_CompressEmptyMessages(t *testing.T) {
	e := NewCompressorEngine(&stubSummarizer{})
	_ = e.Initialize(plugins.ContextEngineConfig{ContextLength: 1000})

	result, err := e.Compress(context.Background(), nil, 0)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCompressorEngine_CompressTooFewMessages(t *testing.T) {
	e := NewCompressorEngine(&stubSummarizer{})
	_ = e.Initialize(plugins.ContextEngineConfig{
		ContextLength: 1000,
		ProtectLastN:  6,
	})

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "hi"},
		{Role: llm.RoleAssistant, Content: "hello"},
	}
	result, err := e.Compress(context.Background(), msgs, 0)
	require.NoError(t, err)
	assert.Equal(t, msgs, result)
}

func TestCompressorEngine_UpdateFromResponse(t *testing.T) {
	e := NewCompressorEngine(&stubSummarizer{})
	_ = e.Initialize(plugins.ContextEngineConfig{ContextLength: 1000, ThresholdPercent: 0.75})

	e.UpdateFromResponse(llm.Usage{
		PromptTokens:     500,
		CompletionTokens: 100,
		TotalTokens:      600,
	})

	status := e.Status()
	assert.Equal(t, 500, status.LastPromptTokens)
	assert.Equal(t, 100, status.LastCompletionTokens)
	assert.Equal(t, 600, status.LastTotalTokens)
}

func TestCompressorEngine_OnSessionReset(t *testing.T) {
	e := NewCompressorEngine(&stubSummarizer{})
	_ = e.Initialize(plugins.ContextEngineConfig{ContextLength: 1000, ThresholdPercent: 0.75})

	e.UpdateFromResponse(llm.Usage{PromptTokens: 500, CompletionTokens: 100, TotalTokens: 600})
	e.OnSessionReset()

	status := e.Status()
	assert.Equal(t, 0, status.LastPromptTokens)
	assert.Equal(t, 0, status.CompressionCount)
}

func TestCompressorEngine_UpdateModel(t *testing.T) {
	e := NewCompressorEngine(&stubSummarizer{})
	_ = e.Initialize(plugins.ContextEngineConfig{
		ContextLength:    1000,
		ThresholdPercent: 0.75,
	})

	assert.Equal(t, 750, e.ThresholdTokens())

	err := e.UpdateModel("gpt-4o-mini", 4000, "", "", "openai")
	require.NoError(t, err)
	assert.Equal(t, 3000, e.ThresholdTokens())
}
