package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- DetectVendor ---

func TestDetectVendor_Empty(t *testing.T) {
	assert.Equal(t, "", DetectVendor(""))
}

func TestDetectVendor_WithSlash(t *testing.T) {
	assert.Equal(t, "anthropic", DetectVendor("anthropic/claude-3-opus"))
	assert.Equal(t, "openai", DetectVendor("openai/gpt-4"))
}

func TestDetectVendor_KnownPrefixes(t *testing.T) {
	cases := map[string]string{
		"claude-3-opus": "anthropic",
		"gpt-4":         "openai",
		"o1-preview":    "openai",
		"o3-mini":       "openai",
		"gemini-pro":    "google",
		"deepseek-chat": "deepseek",
		"llama-2-70b":   "meta-llama",
		"qwen-7b":       "qwen",
		"grok-1":        "x-ai",
	}
	for model, expected := range cases {
		assert.Equal(t, expected, DetectVendor(model), "model: %s", model)
	}
}

func TestDetectVendor_Unknown(t *testing.T) {
	assert.Equal(t, "", DetectVendor("unknown-model"))
}

// --- PrependVendor ---

func TestPrependVendor_AlreadyHasSlash(t *testing.T) {
	assert.Equal(t, "anthropic/claude-3", PrependVendor("anthropic/claude-3"))
}

func TestPrependVendor_KnownModel(t *testing.T) {
	assert.Equal(t, "anthropic/claude-3-opus", PrependVendor("claude-3-opus"))
	assert.Equal(t, "openai/gpt-4", PrependVendor("gpt-4"))
}

func TestPrependVendor_UnknownModel(t *testing.T) {
	assert.Equal(t, "custom-model", PrependVendor("custom-model"))
}

// --- StripVendorPrefix ---

func TestStripVendorPrefix_WithSlash(t *testing.T) {
	assert.Equal(t, "claude-3", StripVendorPrefix("anthropic/claude-3"))
}

func TestStripVendorPrefix_NoSlash(t *testing.T) {
	assert.Equal(t, "gpt-4", StripVendorPrefix("gpt-4"))
}

// --- DotsToHyphens ---

func TestDotsToHyphens(t *testing.T) {
	assert.Equal(t, "claude-3-5-sonnet", DotsToHyphens("claude.3.5.sonnet"))
	assert.Equal(t, "no-dots", DotsToHyphens("no-dots"))
}

// --- StripMatchingProviderPrefix ---

func TestStripMatchingProviderPrefix_NoSlash(t *testing.T) {
	assert.Equal(t, "gpt-4", StripMatchingProviderPrefix("gpt-4", "openai"))
}

func TestStripMatchingProviderPrefix_MatchingPrefix(t *testing.T) {
	assert.Equal(t, "gpt-4", StripMatchingProviderPrefix("openai/gpt-4", "openai"))
}

func TestStripMatchingProviderPrefix_NonMatchingPrefix(t *testing.T) {
	assert.Equal(t, "anthropic/claude-3", StripMatchingProviderPrefix("anthropic/claude-3", "openai"))
}

func TestStripMatchingProviderPrefix_EmptyParts(t *testing.T) {
	assert.Equal(t, "/gpt-4", StripMatchingProviderPrefix("/gpt-4", "openai"))
	assert.Equal(t, "openai/", StripMatchingProviderPrefix("openai/", "openai"))
}

// --- NormalizeModelForProvider ---

func TestNormalizeModelForProvider_Empty(t *testing.T) {
	assert.Equal(t, "", NormalizeModelForProvider("", "openai"))
}

func TestNormalizeModelForProvider_Aggregator(t *testing.T) {
	// Aggregators should prepend vendor.
	assert.Equal(t, "anthropic/claude-3-opus", NormalizeModelForProvider("claude-3-opus", "openrouter"))
	assert.Equal(t, "openai/gpt-4", NormalizeModelForProvider("gpt-4", "nous"))
}

func TestNormalizeModelForProvider_Aggregator_AlreadyPrefixed(t *testing.T) {
	assert.Equal(t, "anthropic/claude-3", NormalizeModelForProvider("anthropic/claude-3", "openrouter"))
}

func TestNormalizeModelForProvider_DotToHyphen(t *testing.T) {
	assert.Equal(t, "claude-3-5-sonnet", NormalizeModelForProvider("claude.3.5.sonnet", "anthropic"))
}

func TestNormalizeModelForProvider_DotToHyphen_StripPrefix(t *testing.T) {
	assert.Equal(t, "claude-3-opus", NormalizeModelForProvider("anthropic/claude-3-opus", "anthropic"))
}

func TestNormalizeModelForProvider_DotToHyphen_WithSlash_NonMatching(t *testing.T) {
	// Non-matching prefix with slash: keep as-is (contains /).
	result := NormalizeModelForProvider("meta-llama/Llama-2", "anthropic")
	assert.Equal(t, "meta-llama/Llama-2", result)
}

func TestNormalizeModelForProvider_StripVendorOnly(t *testing.T) {
	assert.Equal(t, "gpt-4", NormalizeModelForProvider("openai/gpt-4", "copilot"))
}

func TestNormalizeModelForProvider_StripVendorOnly_NoPrefix(t *testing.T) {
	assert.Equal(t, "gpt-4", NormalizeModelForProvider("gpt-4", "copilot"))
}

func TestNormalizeModelForProvider_DeepSeek_Chat(t *testing.T) {
	assert.Equal(t, "deepseek-chat", NormalizeModelForProvider("deepseek-chat", "deepseek"))
}

func TestNormalizeModelForProvider_DeepSeek_Reasoner(t *testing.T) {
	assert.Equal(t, "deepseek-reasoner", NormalizeModelForProvider("deepseek-reasoner", "deepseek"))
}

func TestNormalizeModelForProvider_DeepSeek_InfersChat(t *testing.T) {
	assert.Equal(t, "deepseek-chat", NormalizeModelForProvider("some-model", "deepseek"))
}

func TestNormalizeModelForProvider_DeepSeek_InfersReasoner(t *testing.T) {
	assert.Equal(t, "deepseek-reasoner", NormalizeModelForProvider("deepseek-r1", "deepseek"))
}

func TestNormalizeModelForProvider_DeepSeek_WithMatchingSlash(t *testing.T) {
	// Matching prefix is stripped first, then normalizeForDeepSeek applies.
	assert.Equal(t, "deepseek-chat", NormalizeModelForProvider("deepseek/deepseek-chat", "deepseek"))
}

func TestNormalizeModelForProvider_DeepSeek_WithNonMatchingSlash(t *testing.T) {
	// Non-matching prefix: StripMatchingProviderPrefix returns as-is, contains slash so pass through.
	assert.Equal(t, "other/deepseek-chat", NormalizeModelForProvider("other/deepseek-chat", "deepseek"))
}

func TestNormalizeModelForProvider_MatchingPrefixStrip(t *testing.T) {
	assert.Equal(t, "glm-4", NormalizeModelForProvider("zai/glm-4", "zai"))
}

func TestNormalizeModelForProvider_AuthoritativeNative(t *testing.T) {
	// Authoritative providers keep name as-is.
	assert.Equal(t, "gemini-pro", NormalizeModelForProvider("gemini-pro", "gemini"))
	assert.Equal(t, "vendor/model", NormalizeModelForProvider("vendor/model", "huggingface"))
}

func TestNormalizeModelForProvider_Unknown(t *testing.T) {
	assert.Equal(t, "some-model", NormalizeModelForProvider("some-model", "unknown-provider"))
}
