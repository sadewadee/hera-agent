package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGLMProvider_RequiresAPIKey(t *testing.T) {
	_, err := NewGLMProvider(ProviderConfig{})
	assert.Error(t, err)
}

func TestNewGLMProvider_Success(t *testing.T) {
	p, err := NewGLMProvider(ProviderConfig{APIKey: "glm-key", Model: "glm-4"})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestGLMProvider_ModelInfo(t *testing.T) {
	p, _ := NewGLMProvider(ProviderConfig{APIKey: "k", Model: "glm-4-plus"})
	info := p.ModelInfo()
	assert.Equal(t, "glm", info.Provider)
	assert.Equal(t, "glm-4-plus", info.ID)
	assert.Equal(t, 128000, info.ContextWindow)
	assert.True(t, info.SupportsTools)
}

func TestRegisterGLM(t *testing.T) {
	reg := NewRegistry()
	RegisterGLM(reg)
	p, err := reg.Create("glm", ProviderConfig{APIKey: "k", Model: "test"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
