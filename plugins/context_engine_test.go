package plugins

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalEngine embeds BaseContextEngine and satisfies the remaining required
// methods of ContextEngine (Name, IsAvailable, Initialize, UpdateFromResponse,
// ShouldCompress, Compress).
type minimalEngine struct {
	BaseContextEngine
	name string
}

func (m *minimalEngine) Name() string      { return m.name }
func (m *minimalEngine) IsAvailable() bool { return true }
func (m *minimalEngine) Initialize(cfg ContextEngineConfig) error {
	m.InitBase(cfg)
	return nil
}
func (m *minimalEngine) UpdateFromResponse(usage llm.Usage) {
	m.RecordUsage(usage)
}
func (m *minimalEngine) ShouldCompress(promptTokens int) bool {
	return promptTokens > m.ThresholdTokens()
}
func (m *minimalEngine) Compress(_ context.Context, messages []llm.Message, _ int) ([]llm.Message, error) {
	m.IncrCompressionCount()
	return messages, nil
}

// Verify minimalEngine satisfies the interface at compile time.
var _ ContextEngine = (*minimalEngine)(nil)

func newMinimal(t *testing.T, cfg ContextEngineConfig) *minimalEngine {
	t.Helper()
	e := &minimalEngine{name: "test-engine"}
	require.NoError(t, e.Initialize(cfg))
	return e
}

// --- BaseContextEngine default method tests ---

func TestBaseContextEngine_DefaultPreflightReturnsFalse(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})
	assert.False(t, e.ShouldCompressPreflight(nil))
	assert.False(t, e.ShouldCompressPreflight([]llm.Message{{Role: llm.RoleUser, Content: "hi"}}))
}

func TestBaseContextEngine_OnSessionStartIsNoOp(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})
	assert.NoError(t, e.OnSessionStart("sess-1", "cli", "user-1"))
}

func TestBaseContextEngine_OnSessionEndIsNoOp(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})
	assert.NoError(t, e.OnSessionEnd("sess-1", nil))
}

func TestBaseContextEngine_OnSessionResetClearsCounters(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})

	// Drive some state into the engine.
	e.UpdateFromResponse(llm.Usage{PromptTokens: 500, CompletionTokens: 100, TotalTokens: 600})
	_, _ = e.Compress(context.Background(), nil, 500)

	s := e.Status()
	assert.Equal(t, 500, s.LastPromptTokens)
	assert.Equal(t, 1, s.CompressionCount)

	// Reset must zero those fields.
	e.OnSessionReset()
	s = e.Status()
	assert.Equal(t, 0, s.LastPromptTokens)
	assert.Equal(t, 0, s.LastCompletionTokens)
	assert.Equal(t, 0, s.LastTotalTokens)
	assert.Equal(t, 0, s.CompressionCount)
}

func TestBaseContextEngine_StatusTracksValues(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{
		Name:             "track-test",
		ContextLength:    8192,
		ThresholdPercent: 0.80,
	})

	e.UpdateFromResponse(llm.Usage{PromptTokens: 1000, CompletionTokens: 200, TotalTokens: 1200})
	s := e.Status()

	assert.Equal(t, "track-test", s.Name)
	assert.Equal(t, 1000, s.LastPromptTokens)
	assert.Equal(t, 200, s.LastCompletionTokens)
	assert.Equal(t, 1200, s.LastTotalTokens)
	assert.Equal(t, 8192, s.ContextLength)
	threshold := 0.80
	expectedThreshold := int(float64(8192) * threshold)
	assert.Equal(t, expectedThreshold, s.ThresholdTokens)
	assert.InDelta(t, float64(1000)/float64(expectedThreshold)*100, s.UsagePercent, 0.01)
}

func TestBaseContextEngine_GetToolSchemasReturnsNil(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})
	assert.Nil(t, e.GetToolSchemas())
}

func TestBaseContextEngine_HandleToolCallReturnsError(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})
	_, err := e.HandleToolCall(context.Background(), "unknown_tool", json.RawMessage(`{}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown_tool")
}

func TestBaseContextEngine_UpdateModelRecalculatesThreshold(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{
		ContextLength:    4096,
		ThresholdPercent: 0.75,
	})
	assert.Equal(t, int(4096*0.75), e.ThresholdTokens())

	// Switch to a larger model.
	require.NoError(t, e.UpdateModel("gpt-4o-128k", 131072, "", "", "openai"))
	assert.Equal(t, int(131072*0.75), e.ThresholdTokens())
}

func TestBaseContextEngine_ThresholdDefaultsTo75Percent(t *testing.T) {
	// ThresholdPercent == 0 should default to 0.75.
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096, ThresholdPercent: 0})
	assert.Equal(t, int(4096*0.75), e.ThresholdTokens())
}

func TestBaseContextEngine_ProtectDefaults(t *testing.T) {
	// Both ProtectFirstN and ProtectLastN default to sane values when zero.
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})
	assert.Equal(t, 3, e.ProtectFirstN())
	assert.Equal(t, 6, e.ProtectLastN())
}

func TestBaseContextEngine_ShutdownIsNoOp(t *testing.T) {
	e := newMinimal(t, ContextEngineConfig{ContextLength: 4096})
	// Must not panic.
	e.Shutdown()
}

// --- Registry integration with new interface ---

func TestRegistry_RegisterContextEngine_NewInterface(t *testing.T) {
	reg := NewRegistry()
	e := &minimalEngine{name: "new-iface"}
	require.NoError(t, e.Initialize(ContextEngineConfig{ContextLength: 4096}))

	reg.RegisterContextEngine(e)
	got := reg.GetContextEngine("new-iface")
	require.NotNil(t, got)
	assert.Equal(t, "new-iface", got.Name())
	assert.True(t, got.IsAvailable())
}
