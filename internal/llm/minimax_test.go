package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMiniMaxProvider_RequiresAPIKey(t *testing.T) {
	_, err := NewMiniMaxProvider(ProviderConfig{})
	assert.Error(t, err)
}

func TestNewMiniMaxProvider_Success(t *testing.T) {
	p, err := NewMiniMaxProvider(ProviderConfig{APIKey: "mm-key", Model: "abab6"})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestMiniMaxProvider_ModelInfo(t *testing.T) {
	p, _ := NewMiniMaxProvider(ProviderConfig{APIKey: "k", Model: "abab6-chat"})
	info := p.ModelInfo()
	assert.Equal(t, "minimax", info.Provider)
	assert.Equal(t, 128000, info.ContextWindow)
	assert.True(t, info.SupportsTools)
}

func TestRegisterMiniMax(t *testing.T) {
	reg := NewRegistry()
	RegisterMiniMax(reg)
	p, err := reg.Create("minimax", ProviderConfig{APIKey: "k", Model: "test"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
