package hcore

import (
	"fmt"
	"log/slog"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/llm"
)

// BuildLLMProvider constructs the primary LLM provider plus any configured
// fallbacks and returns the result wrapped in a FallbackProvider so callers
// see a single Provider. When fallback_providers is empty, the wrapper is
// effectively a pass-through.
//
// Provider config resolution for each name:
//   - BaseURL comes from cfg.Provider[name].BaseURL
//   - APIKey comes from config.ResolveAPIKey (config → env var)
//   - APIKeys comes from cfg.Provider[name].APIKeys (credential pool)
//   - Model comes from cfg.Agent.DefaultModel (same model across all
//     providers — call-site picks one, fallbacks mirror)
//
// Any fallback that fails to construct is skipped with a warning; the
// primary must succeed or BuildLLMProvider returns an error.
func BuildLLMProvider(cfg *config.Config, registry *llm.Registry) (llm.Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	primaryName := cfg.Agent.DefaultProvider
	if primaryName == "" {
		return nil, fmt.Errorf("agent.default_provider not set")
	}

	primary, err := buildOne(cfg, registry, primaryName)
	if err != nil {
		return nil, fmt.Errorf("build primary provider %q: %w", primaryName, err)
	}

	var fallbacks []llm.Provider
	var fallbackLabels []string
	for _, name := range cfg.Agent.FallbackProviders {
		if name == primaryName {
			continue // don't list primary as its own fallback
		}
		p, err := buildOne(cfg, registry, name)
		if err != nil {
			slog.Warn("llm: skipping fallback provider", "name", name, "error", err)
			continue
		}
		fallbacks = append(fallbacks, p)
		fallbackLabels = append(fallbackLabels, name)
	}

	if len(fallbacks) == 0 {
		return primary, nil
	}
	return llm.NewFallbackProvider(primary, primaryName, fallbacks, fallbackLabels), nil
}

func buildOne(cfg *config.Config, registry *llm.Registry, name string) (llm.Provider, error) {
	providerCfg := llm.ProviderConfig{
		Model: cfg.Agent.DefaultModel,
	}
	if entry, ok := cfg.Provider[name]; ok {
		if entry.BaseURL != "" {
			providerCfg.BaseURL = entry.BaseURL
		}
		providerCfg.APIKeys = entry.APIKeys
	}
	providerCfg.APIKey = config.ResolveAPIKey(cfg, name)

	// Allow keyless providers (ollama, local compatible servers).
	if providerCfg.APIKey == "" && len(providerCfg.APIKeys) == 0 &&
		name != "ollama" && name != "compatible" {
		return nil, fmt.Errorf("no API key configured for %q", name)
	}
	return registry.Create(name, providerCfg)
}
