// Package cli provides the Hera CLI application.
//
// model_switch.go implements the shared model-switching logic for CLI and
// gateway /model commands: parse flags, alias resolution, provider resolution,
// credential resolution, model normalization, metadata lookup.
package cli

import (
	"log/slog"
	"strings"
)

// ModelIdentity maps an alias to a vendor and family prefix.
type ModelIdentity struct {
	Vendor string
	Family string
}

// ModelAliases maps short names to (vendor, family) identities.
var ModelAliases = map[string]ModelIdentity{
	"sonnet":   {Vendor: "anthropic", Family: "claude-sonnet"},
	"opus":     {Vendor: "anthropic", Family: "claude-opus"},
	"haiku":    {Vendor: "anthropic", Family: "claude-haiku"},
	"claude":   {Vendor: "anthropic", Family: "claude"},
	"gpt5":     {Vendor: "openai", Family: "gpt-5"},
	"gpt":      {Vendor: "openai", Family: "gpt"},
	"codex":    {Vendor: "openai", Family: "codex"},
	"o3":       {Vendor: "openai", Family: "o3"},
	"o4":       {Vendor: "openai", Family: "o4"},
	"gemini":   {Vendor: "google", Family: "gemini"},
	"deepseek": {Vendor: "deepseek", Family: "deepseek-chat"},
	"grok":     {Vendor: "x-ai", Family: "grok"},
	"llama":    {Vendor: "meta-llama", Family: "llama"},
	"qwen":     {Vendor: "qwen", Family: "qwen"},
	"minimax":  {Vendor: "minimax", Family: "minimax"},
	"nemotron": {Vendor: "nvidia", Family: "nemotron"},
	"kimi":     {Vendor: "moonshotai", Family: "kimi"},
	"glm":      {Vendor: "z-ai", Family: "glm"},
	"step":     {Vendor: "stepfun", Family: "step"},
	"mimo":     {Vendor: "xiaomi", Family: "mimo"},
	"trinity":  {Vendor: "arcee-ai", Family: "trinity"},
}

// DirectAlias is an exact model mapping that bypasses catalog resolution.
type DirectAlias struct {
	Model    string
	Provider string
	BaseURL  string
}

// DirectAliases holds merged built-in and user-config direct aliases.
var DirectAliases = map[string]DirectAlias{}

// ModelSwitchResult holds the result of a model switch attempt.
type ModelSwitchResult struct {
	Success          bool
	NewModel         string
	TargetProvider   string
	ProviderChanged  bool
	APIKey           string
	BaseURL          string
	APIMode          string
	ErrorMessage     string
	WarningMessage   string
	ProviderLabel    string
	ResolvedViaAlias string
	IsGlobal         bool
}

// ParseModelFlags parses --provider and --global flags from /model args.
func ParseModelFlags(rawArgs string) (modelInput, explicitProvider string, isGlobal bool) {
	if strings.Contains(rawArgs, "--global") {
		isGlobal = true
		rawArgs = strings.ReplaceAll(rawArgs, "--global", "")
		rawArgs = strings.TrimSpace(rawArgs)
	}

	parts := strings.Fields(rawArgs)
	var filtered []string
	for i := 0; i < len(parts); i++ {
		if parts[i] == "--provider" && i+1 < len(parts) {
			explicitProvider = parts[i+1]
			i++ // skip next
		} else {
			filtered = append(filtered, parts[i])
		}
	}

	modelInput = strings.TrimSpace(strings.Join(filtered, " "))
	return
}

// ResolveAlias resolves a short alias against a provider's catalog.
func ResolveAlias(rawInput, currentProvider string) (provider, resolvedModel, aliasName string, found bool) {
	key := strings.TrimSpace(strings.ToLower(rawInput))

	// Check direct aliases first.
	if da, ok := DirectAliases[key]; ok {
		return da.Provider, da.Model, key, true
	}

	// Reverse lookup.
	for name, da := range DirectAliases {
		if strings.ToLower(da.Model) == key {
			return da.Provider, da.Model, name, true
		}
	}

	identity, ok := ModelAliases[key]
	if !ok {
		return "", "", "", false
	}

	slog.Debug("alias resolved",
		"alias", key,
		"vendor", identity.Vendor,
		"family", identity.Family,
		"provider", currentProvider,
	)

	// Catalog lookup would happen here in full implementation.
	return currentProvider, rawInput, key, false
}

// SwitchModel implements the core model-switching pipeline.
func SwitchModel(
	rawInput, currentProvider, currentModel, currentBaseURL, currentAPIKey string,
	isGlobal bool,
	explicitProvider string,
) ModelSwitchResult {
	newModel := strings.TrimSpace(rawInput)
	targetProvider := currentProvider
	resolvedAlias := ""
	providerChanged := false

	// PATH A: Explicit --provider given.
	if explicitProvider != "" {
		targetProvider = explicitProvider
		providerChanged = targetProvider != currentProvider

		// Resolve alias on target provider.
		if p, m, alias, found := ResolveAlias(newModel, targetProvider); found {
			targetProvider = p
			newModel = m
			resolvedAlias = alias
		}
	} else {
		// PATH B: No explicit provider.
		if p, m, alias, found := ResolveAlias(rawInput, currentProvider); found {
			targetProvider = p
			newModel = m
			resolvedAlias = alias
		}

		providerChanged = targetProvider != currentProvider
	}

	// Normalize model name.
	newModel = NormalizeModelForProvider(newModel, targetProvider)

	// Check for non-agentic model warning.
	var warning string
	if strings.Contains(strings.ToLower(newModel), "hermes") {
		warning = "The selected model family is NOT agentic and lacks tool-calling capabilities."
	}

	return ModelSwitchResult{
		Success:          true,
		NewModel:         newModel,
		TargetProvider:   targetProvider,
		ProviderChanged:  providerChanged,
		APIKey:           currentAPIKey,
		BaseURL:          currentBaseURL,
		WarningMessage:   warning,
		ResolvedViaAlias: resolvedAlias,
		IsGlobal:         isGlobal,
	}
}

// NonAgenticModelWarning is shown when the user selects a model family that
// lacks tool-calling capabilities required for agent workflows.
const NonAgenticModelWarning = "The selected model family is NOT agentic and is not designed " +
	"for use with Hera Agent. It lacks the tool-calling capabilities " +
	"required for agent workflows. Consider using an agentic model instead " +
	"(Claude, GPT, Gemini, DeepSeek, etc.)."
