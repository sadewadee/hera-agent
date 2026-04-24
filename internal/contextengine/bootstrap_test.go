package contextengine

import (
	"context"
	"testing"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSummarizer struct{}

func (f *fakeSummarizer) Summarize(_ context.Context, _ []llm.Message) (string, error) {
	return "summary", nil
}

func TestRegisterBuiltinEngines(t *testing.T) {
	reg := plugins.NewRegistry()
	RegisterBuiltinEngines(reg, &fakeSummarizer{})

	eng := reg.GetContextEngine("compressor")
	require.NotNil(t, eng)
	assert.Equal(t, "compressor", eng.Name())
	assert.True(t, eng.IsAvailable())
}

func TestNewFromConfig_Compressor(t *testing.T) {
	reg := plugins.NewRegistry()
	RegisterBuiltinEngines(reg, &fakeSummarizer{})

	cfg := config.AgentConfig{}
	cfg.Compression.Enabled = true
	cfg.Compression.Engine = "compressor"
	cfg.Compression.Threshold = 0.75
	cfg.Compression.ProtectedTurns = 4

	modelInfo := llm.ModelMetadata{ContextWindow: 8000}

	eng, err := NewFromConfig(cfg, modelInfo, reg)
	require.NoError(t, err)
	assert.Equal(t, "compressor", eng.Name())

	status := eng.Status()
	assert.Equal(t, 6000, status.ThresholdTokens)
	assert.Equal(t, 8000, status.ContextLength)
}

func TestNewFromConfig_DefaultEngine(t *testing.T) {
	reg := plugins.NewRegistry()
	RegisterBuiltinEngines(reg, &fakeSummarizer{})

	cfg := config.AgentConfig{}
	cfg.Compression.Enabled = true
	// Engine field empty — should default to "compressor"

	modelInfo := llm.ModelMetadata{ContextWindow: 4000}

	eng, err := NewFromConfig(cfg, modelInfo, reg)
	require.NoError(t, err)
	assert.Equal(t, "compressor", eng.Name())
}

func TestNewFromConfig_UnknownEngine(t *testing.T) {
	reg := plugins.NewRegistry()
	RegisterBuiltinEngines(reg, &fakeSummarizer{})

	cfg := config.AgentConfig{}
	cfg.Compression.Engine = "nonexistent"

	_, err := NewFromConfig(cfg, llm.ModelMetadata{ContextWindow: 4000}, reg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown context engine")
	assert.Contains(t, err.Error(), "compressor")
}

func TestRegisterBuiltinEngines_ListContainsCompressor(t *testing.T) {
	reg := plugins.NewRegistry()
	RegisterBuiltinEngines(reg, &fakeSummarizer{})

	list := reg.ListContextEngines()
	require.Len(t, list, 1)
	assert.Equal(t, "compressor", list[0].Name)
	assert.True(t, list[0].Available)
}
