package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKimiProvider_RequiresAPIKey(t *testing.T) {
	_, err := NewKimiProvider(ProviderConfig{})
	assert.Error(t, err)
}

func TestNewKimiProvider_Success(t *testing.T) {
	p, err := NewKimiProvider(ProviderConfig{APIKey: "kimi-key", Model: "moonshot-v1"})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestKimiProvider_ModelInfo(t *testing.T) {
	p, _ := NewKimiProvider(ProviderConfig{APIKey: "k", Model: "moonshot-v1-128k"})
	info := p.ModelInfo()
	assert.Equal(t, "kimi", info.Provider)
	assert.Equal(t, "moonshot-v1-128k", info.ID)
	assert.Equal(t, 128000, info.ContextWindow)
}

func TestKimiProvider_CustomBaseURL(t *testing.T) {
	p, err := NewKimiProvider(ProviderConfig{
		APIKey:  "k",
		BaseURL: "https://custom.moonshot.cn/v1/",
		Model:   "test",
	})
	require.NoError(t, err)
	kp := p.(*KimiProvider)
	assert.Equal(t, "https://custom.moonshot.cn/v1", kp.CompatibleProvider.baseURL)
}

func TestRegisterKimi(t *testing.T) {
	reg := NewRegistry()
	RegisterKimi(reg)
	p, err := reg.Create("kimi", ProviderConfig{APIKey: "k", Model: "test"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
