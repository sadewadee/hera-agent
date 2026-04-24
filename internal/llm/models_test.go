package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupModel_Exists(t *testing.T) {
	meta, ok := LookupModel("gpt-4o")
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", meta.ID)
	assert.Equal(t, "openai", meta.Provider)
	assert.Equal(t, 128000, meta.ContextWindow)
	assert.True(t, meta.SupportsTools)
	assert.True(t, meta.SupportsVision)
}

func TestLookupModel_NotFound(t *testing.T) {
	_, ok := LookupModel("nonexistent-model")
	assert.False(t, ok)
}

func TestLookupModel_Anthropic(t *testing.T) {
	meta, ok := LookupModel("claude-3-5-sonnet-20241022")
	require.True(t, ok)
	assert.Equal(t, "anthropic", meta.Provider)
	assert.Equal(t, 200000, meta.ContextWindow)
}

func TestLookupModel_Gemini(t *testing.T) {
	meta, ok := LookupModel("gemini-2.0-flash")
	require.True(t, ok)
	assert.Equal(t, "gemini", meta.Provider)
	assert.Equal(t, 1048576, meta.ContextWindow)
}

func TestLookupModel_Ollama(t *testing.T) {
	meta, ok := LookupModel("llama3.1")
	require.True(t, ok)
	assert.Equal(t, "ollama", meta.Provider)
	assert.Equal(t, float64(0), meta.CostPer1kIn)
}

func TestModelsByProvider_OpenAI(t *testing.T) {
	models := ModelsByProvider("openai")
	assert.NotEmpty(t, models)
	for _, m := range models {
		assert.Equal(t, "openai", m.Provider)
	}
}

func TestModelsByProvider_Anthropic(t *testing.T) {
	models := ModelsByProvider("anthropic")
	assert.NotEmpty(t, models)
	for _, m := range models {
		assert.Equal(t, "anthropic", m.Provider)
	}
}

func TestModelsByProvider_Empty(t *testing.T) {
	models := ModelsByProvider("nonexistent_provider")
	assert.Empty(t, models)
}

func TestAllProviders(t *testing.T) {
	providers := AllProviders()
	assert.NotEmpty(t, providers)

	provSet := make(map[string]bool)
	for _, p := range providers {
		provSet[p] = true
	}
	assert.True(t, provSet["openai"])
	assert.True(t, provSet["anthropic"])
	assert.True(t, provSet["gemini"])
	assert.True(t, provSet["ollama"])
}

func TestAllProviders_NoDuplicates(t *testing.T) {
	providers := AllProviders()
	seen := make(map[string]bool)
	for _, p := range providers {
		assert.False(t, seen[p], "duplicate provider: %s", p)
		seen[p] = true
	}
}

func TestKnownModels_AllHaveRequiredFields(t *testing.T) {
	for id, meta := range KnownModels {
		assert.NotEmpty(t, meta.ID, "model %s missing ID", id)
		assert.Equal(t, id, meta.ID, "model key %s doesn't match ID %s", id, meta.ID)
		assert.NotEmpty(t, meta.Provider, "model %s missing Provider", id)
		assert.Greater(t, meta.ContextWindow, 0, "model %s has zero context window", id)
		assert.Greater(t, meta.MaxOutput, 0, "model %s has zero max output", id)
	}
}
