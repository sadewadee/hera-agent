package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- ParseModelFlags ---

func TestParseModelFlags_Simple(t *testing.T) {
	model, provider, global := ParseModelFlags("gpt-4")
	assert.Equal(t, "gpt-4", model)
	assert.Empty(t, provider)
	assert.False(t, global)
}

func TestParseModelFlags_WithGlobal(t *testing.T) {
	model, provider, global := ParseModelFlags("gpt-4 --global")
	assert.Equal(t, "gpt-4", model)
	assert.Empty(t, provider)
	assert.True(t, global)
}

func TestParseModelFlags_WithProvider(t *testing.T) {
	model, provider, global := ParseModelFlags("gpt-4 --provider openai")
	assert.Equal(t, "gpt-4", model)
	assert.Equal(t, "openai", provider)
	assert.False(t, global)
}

func TestParseModelFlags_WithBothFlags(t *testing.T) {
	model, provider, global := ParseModelFlags("claude-3 --provider anthropic --global")
	assert.Equal(t, "claude-3", model)
	assert.Equal(t, "anthropic", provider)
	assert.True(t, global)
}

func TestParseModelFlags_Empty(t *testing.T) {
	model, provider, global := ParseModelFlags("")
	assert.Empty(t, model)
	assert.Empty(t, provider)
	assert.False(t, global)
}

// --- ResolveAlias ---

func TestResolveAlias_DirectAlias(t *testing.T) {
	DirectAliases["test-alias"] = DirectAlias{Model: "test-model", Provider: "test-provider"}
	defer delete(DirectAliases, "test-alias")

	provider, model, alias, found := ResolveAlias("test-alias", "openai")
	assert.True(t, found)
	assert.Equal(t, "test-provider", provider)
	assert.Equal(t, "test-model", model)
	assert.Equal(t, "test-alias", alias)
}

func TestResolveAlias_ModelAliasLookup(t *testing.T) {
	_, _, _, found := ResolveAlias("sonnet", "openrouter")
	// ModelAliases has "sonnet" mapped, but ResolveAlias returns found=false
	// because catalog lookup is not implemented. The identity is still set.
	assert.False(t, found)
}

func TestResolveAlias_Unknown(t *testing.T) {
	_, _, _, found := ResolveAlias("nonexistent-model", "openai")
	assert.False(t, found)
}

// --- SwitchModel ---

func TestSwitchModel_Simple(t *testing.T) {
	result := SwitchModel("gpt-4", "openai", "gpt-3.5", "", "sk-key", false, "")
	assert.True(t, result.Success)
	assert.Equal(t, "gpt-4", result.NewModel)
	assert.Equal(t, "openai", result.TargetProvider)
	assert.False(t, result.ProviderChanged)
}

func TestSwitchModel_ExplicitProvider(t *testing.T) {
	result := SwitchModel("claude-3", "openai", "gpt-4", "", "sk-key", false, "anthropic")
	assert.True(t, result.Success)
	assert.Equal(t, "anthropic", result.TargetProvider)
	assert.True(t, result.ProviderChanged)
}

func TestSwitchModel_Global(t *testing.T) {
	result := SwitchModel("gpt-4", "openai", "gpt-3.5", "", "sk-key", true, "")
	assert.True(t, result.IsGlobal)
}

func TestSwitchModel_NonAgenticWarning(t *testing.T) {
	result := SwitchModel("hermes-3-llama", "openai", "gpt-4", "", "key", false, "")
	assert.Contains(t, result.WarningMessage, "NOT agentic")
}

func TestSwitchModel_NoNonAgenticWarning(t *testing.T) {
	result := SwitchModel("gpt-4", "openai", "gpt-3.5", "", "key", false, "")
	assert.Empty(t, result.WarningMessage)
}

// --- ModelAliases ---

func TestModelAliases_KnownEntries(t *testing.T) {
	assert.Equal(t, "anthropic", ModelAliases["sonnet"].Vendor)
	assert.Equal(t, "openai", ModelAliases["gpt"].Vendor)
	assert.Equal(t, "google", ModelAliases["gemini"].Vendor)
	assert.Equal(t, "deepseek", ModelAliases["deepseek"].Vendor)
}

// --- NonAgenticModelWarning ---

func TestNonAgenticModelWarning_Constant(t *testing.T) {
	assert.Contains(t, NonAgenticModelWarning, "NOT agentic")
	assert.Contains(t, NonAgenticModelWarning, "tool-calling")
}
