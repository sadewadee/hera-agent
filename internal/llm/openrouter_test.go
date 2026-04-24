package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenRouterProvider_RequiresAPIKey(t *testing.T) {
	_, err := NewOpenRouterProvider(ProviderConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}

func TestNewOpenRouterProvider_Success(t *testing.T) {
	p, err := NewOpenRouterProvider(ProviderConfig{
		APIKey: "test-key",
		Model:  "anthropic/claude-3-opus",
	})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestOpenRouterProvider_ModelInfo(t *testing.T) {
	p, _ := NewOpenRouterProvider(ProviderConfig{
		APIKey: "test-key",
		Model:  "anthropic/claude-3-opus",
	})
	info := p.ModelInfo()
	assert.Equal(t, "anthropic/claude-3-opus", info.ID)
	assert.Equal(t, "openrouter", info.Provider)
	assert.True(t, info.SupportsTools)
	assert.Equal(t, 128000, info.ContextWindow)
}

func TestOpenRouterProvider_CustomBaseURL(t *testing.T) {
	p, err := NewOpenRouterProvider(ProviderConfig{
		APIKey:  "test-key",
		BaseURL: "https://custom.openrouter.ai/v1/",
		Model:   "test-model",
	})
	require.NoError(t, err)
	or := p.(*OpenRouterProvider)
	assert.Equal(t, "https://custom.openrouter.ai/v1", or.CompatibleProvider.baseURL)
}

func TestRegisterOpenRouter(t *testing.T) {
	reg := NewRegistry()
	RegisterOpenRouter(reg)
	p, err := reg.Create("openrouter", ProviderConfig{APIKey: "test-key", Model: "test"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
