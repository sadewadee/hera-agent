// Package cli provides the Hera CLI application.
//
// model_normalize.go implements per-provider model name normalization.
// Different LLM providers expect model identifiers in different formats.
package cli

import "strings"

// VendorPrefixes maps the first hyphen-delimited token to a vendor slug.
var VendorPrefixes = map[string]string{
	"claude":   "anthropic",
	"gpt":      "openai",
	"o1":       "openai",
	"o3":       "openai",
	"o4":       "openai",
	"gemini":   "google",
	"gemma":    "google",
	"deepseek": "deepseek",
	"glm":      "z-ai",
	"kimi":     "moonshotai",
	"minimax":  "minimax",
	"grok":     "x-ai",
	"qwen":     "qwen",
	"mimo":     "xiaomi",
	"nemotron": "nvidia",
	"llama":    "meta-llama",
	"step":     "stepfun",
	"trinity":  "arcee-ai",
}

// Provider category sets.
var (
	AggregatorProviders = map[string]bool{
		"openrouter": true, "nous": true, "ai-gateway": true, "kilocode": true,
	}
	DotToHyphenProviders = map[string]bool{
		"anthropic": true, "opencode-zen": true,
	}
	StripVendorOnlyProviders = map[string]bool{
		"copilot": true, "copilot-acp": true, "openai-codex": true,
	}
	AuthoritativeNativeProviders = map[string]bool{
		"gemini": true, "huggingface": true,
	}
	MatchingPrefixStripProviders = map[string]bool{
		"zai": true, "kimi-coding": true, "minimax": true, "minimax-cn": true,
		"alibaba": true, "qwen-oauth": true, "xiaomi": true, "custom": true,
	}
)

// DeepSeek canonical model names.
var (
	DeepSeekReasonerKeywords = map[string]bool{
		"reasoner": true, "r1": true, "think": true, "reasoning": true, "cot": true,
	}
	DeepSeekCanonicalModels = map[string]bool{
		"deepseek-chat": true, "deepseek-reasoner": true,
	}
)

// NormalizeModelForProvider translates a model name into the format the target
// provider's API expects.
func NormalizeModelForProvider(modelInput, targetProvider string) string {
	name := strings.TrimSpace(modelInput)
	if name == "" {
		return name
	}
	provider := strings.TrimSpace(strings.ToLower(targetProvider))

	if AggregatorProviders[provider] {
		return PrependVendor(name)
	}
	if DotToHyphenProviders[provider] {
		bare := StripMatchingProviderPrefix(name, provider)
		if strings.Contains(bare, "/") {
			return bare
		}
		return DotsToHyphens(bare)
	}
	if StripVendorOnlyProviders[provider] {
		stripped := StripMatchingProviderPrefix(name, provider)
		if stripped == name && strings.HasPrefix(name, "openai/") {
			return name[len("openai/"):]
		}
		return stripped
	}
	if provider == "deepseek" {
		bare := StripMatchingProviderPrefix(name, provider)
		if strings.Contains(bare, "/") {
			return bare
		}
		return normalizeForDeepSeek(bare)
	}
	if MatchingPrefixStripProviders[provider] {
		return StripMatchingProviderPrefix(name, provider)
	}
	if AuthoritativeNativeProviders[provider] {
		return name
	}
	return name
}

// DetectVendor detects the vendor slug from a model name.
func DetectVendor(modelName string) string {
	name := strings.TrimSpace(modelName)
	if name == "" {
		return ""
	}
	if strings.Contains(name, "/") {
		return strings.ToLower(strings.SplitN(name, "/", 2)[0])
	}
	nameLower := strings.ToLower(name)
	firstToken := strings.SplitN(nameLower, "-", 2)[0]
	if vendor, ok := VendorPrefixes[firstToken]; ok {
		return vendor
	}
	for prefix, vendor := range VendorPrefixes {
		if strings.HasPrefix(nameLower, prefix) {
			return vendor
		}
	}
	return ""
}

// PrependVendor adds the detected vendor/ prefix if missing.
func PrependVendor(modelName string) string {
	if strings.Contains(modelName, "/") {
		return modelName
	}
	vendor := DetectVendor(modelName)
	if vendor != "" {
		return vendor + "/" + modelName
	}
	return modelName
}

// StripVendorPrefix removes a vendor/ prefix if present.
func StripVendorPrefix(modelName string) string {
	if idx := strings.Index(modelName, "/"); idx >= 0 {
		return modelName[idx+1:]
	}
	return modelName
}

// DotsToHyphens replaces dots with hyphens in a model name.
func DotsToHyphens(modelName string) string {
	return strings.ReplaceAll(modelName, ".", "-")
}

// StripMatchingProviderPrefix strips vendor/ only when the prefix matches.
func StripMatchingProviderPrefix(modelName, targetProvider string) string {
	if !strings.Contains(modelName, "/") {
		return modelName
	}
	parts := strings.SplitN(modelName, "/", 2)
	prefix := strings.TrimSpace(strings.ToLower(parts[0]))
	remainder := strings.TrimSpace(parts[1])
	if prefix == "" || remainder == "" {
		return modelName
	}
	normalizedTarget := strings.TrimSpace(strings.ToLower(targetProvider))
	if prefix == normalizedTarget {
		return remainder
	}
	return modelName
}

func normalizeForDeepSeek(modelName string) string {
	bare := strings.ToLower(StripVendorPrefix(modelName))
	if DeepSeekCanonicalModels[bare] {
		return bare
	}
	for keyword := range DeepSeekReasonerKeywords {
		if strings.Contains(bare, keyword) {
			return "deepseek-reasoner"
		}
	}
	return "deepseek-chat"
}
