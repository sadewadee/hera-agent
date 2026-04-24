package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHuggingFaceProvider_RequiresAPIKey(t *testing.T) {
	_, err := NewHuggingFaceProvider(ProviderConfig{})
	assert.Error(t, err)
}

func TestNewHuggingFaceProvider_Success(t *testing.T) {
	p, err := NewHuggingFaceProvider(ProviderConfig{APIKey: "hf-test", Model: "meta-llama/Llama-2"})
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestHuggingFaceProvider_ModelInfo(t *testing.T) {
	p, _ := NewHuggingFaceProvider(ProviderConfig{APIKey: "hf-test", Model: "meta-llama/Llama-2"})
	info := p.ModelInfo()
	assert.Equal(t, "huggingface", info.Provider)
	assert.Equal(t, "meta-llama/Llama-2", info.ID)
	assert.Equal(t, 32000, info.ContextWindow)
}

func TestHuggingFaceProvider_CustomBaseURL(t *testing.T) {
	p, err := NewHuggingFaceProvider(ProviderConfig{
		APIKey:  "hf-test",
		BaseURL: "https://my-tgi-server.com/v1/",
		Model:   "test",
	})
	require.NoError(t, err)
	hf := p.(*HuggingFaceProvider)
	assert.Equal(t, "https://my-tgi-server.com/v1", hf.CompatibleProvider.baseURL)
}

func TestRegisterHuggingFace(t *testing.T) {
	reg := NewRegistry()
	RegisterHuggingFace(reg)
	p, err := reg.Create("huggingface", ProviderConfig{APIKey: "k", Model: "test"})
	require.NoError(t, err)
	assert.NotNil(t, p)
}
