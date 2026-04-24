package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexModels_NotEmpty(t *testing.T) {
	assert.Greater(t, len(CodexModels), 0)
}

func TestCodexModels_HaveValidFields(t *testing.T) {
	for _, m := range CodexModels {
		assert.NotEmpty(t, m.ID, "model ID should not be empty")
		assert.NotEmpty(t, m.Name, "model Name should not be empty")
		assert.Greater(t, m.ContextWindow, 0, "model %s context window should be positive", m.ID)
	}
}

func TestFindModel_Found(t *testing.T) {
	m := FindModel("gpt-4o")
	require.NotNil(t, m)
	assert.Equal(t, "GPT-4o", m.Name)
	assert.Equal(t, 128000, m.ContextWindow)
}

func TestFindModel_NotFound(t *testing.T) {
	m := FindModel("nonexistent-model")
	assert.Nil(t, m)
}

func TestFindModel_AllIDsResolvable(t *testing.T) {
	for _, m := range CodexModels {
		found := FindModel(m.ID)
		require.NotNil(t, found, "FindModel should find %s", m.ID)
		assert.Equal(t, m.ID, found.ID)
	}
}

func TestFindModel_GPT35Turbo(t *testing.T) {
	m := FindModel("gpt-3.5-turbo")
	require.NotNil(t, m)
	assert.Equal(t, 16385, m.ContextWindow)
}

func TestFindModel_Claude3Opus(t *testing.T) {
	m := FindModel("claude-3-opus")
	require.NotNil(t, m)
	assert.Equal(t, 200000, m.ContextWindow)
}
