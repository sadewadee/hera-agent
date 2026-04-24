package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMistralProvider_RequiresAPIKey(t *testing.T) {
	_, err := NewMistralProvider(ProviderConfig{})
	assert.Error(t, err)
}

func TestNewMistralProvider_Success(t *testing.T) {
	p, err := NewMistralProvider(ProviderConfig{APIKey: "test", Model: "mistral-large"})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestMistralProvider_ModelInfo_Large(t *testing.T) {
	p, _ := NewMistralProvider(ProviderConfig{APIKey: "k", Model: "mistral-large-latest"})
	info := p.ModelInfo()
	assert.Equal(t, "mistral", info.Provider)
	assert.Equal(t, 128000, info.ContextWindow)
	assert.True(t, info.SupportsTools)
}

func TestMistralProvider_ModelInfo_Small(t *testing.T) {
	p, _ := NewMistralProvider(ProviderConfig{APIKey: "k", Model: "mistral-small-latest"})
	info := p.ModelInfo()
	assert.Equal(t, 32000, info.ContextWindow)
}

func TestMistralProvider_ModelInfo_Codestral(t *testing.T) {
	p, _ := NewMistralProvider(ProviderConfig{APIKey: "k", Model: "codestral-latest"})
	info := p.ModelInfo()
	assert.Equal(t, 32000, info.ContextWindow)
}

func TestMistralProvider_ModelInfo_Default(t *testing.T) {
	p, _ := NewMistralProvider(ProviderConfig{APIKey: "k", Model: "some-unknown"})
	info := p.ModelInfo()
	assert.Equal(t, 32000, info.ContextWindow)
}

func TestRegisterMistral(t *testing.T) {
	reg := NewRegistry()
	RegisterMistral(reg)
	p, err := reg.Create("mistral", ProviderConfig{APIKey: "k", Model: "test"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
