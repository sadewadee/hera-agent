package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNousProvider_RequiresAPIKey(t *testing.T) {
	_, err := NewNousProvider(ProviderConfig{})
	assert.Error(t, err)
}

func TestNewNousProvider_Success(t *testing.T) {
	p, err := NewNousProvider(ProviderConfig{APIKey: "nous-key", Model: "hermes-3"})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestNousProvider_ModelInfo(t *testing.T) {
	p, _ := NewNousProvider(ProviderConfig{APIKey: "k", Model: "hermes-3-llama"})
	info := p.ModelInfo()
	assert.Equal(t, "nous", info.Provider)
	assert.Equal(t, "hermes-3-llama", info.ID)
	assert.Equal(t, 128000, info.ContextWindow)
}

func TestNousProvider_CustomBaseURL(t *testing.T) {
	p, err := NewNousProvider(ProviderConfig{
		APIKey:  "k",
		BaseURL: "https://my-nous.com/v1/",
		Model:   "test",
	})
	require.NoError(t, err)
	nous := p.(*NousProvider)
	assert.Equal(t, "https://my-nous.com/v1", nous.CompatibleProvider.baseURL)
}

func TestRegisterNous(t *testing.T) {
	reg := NewRegistry()
	RegisterNous(reg)
	p, err := reg.Create("nous", ProviderConfig{APIKey: "k", Model: "test"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
