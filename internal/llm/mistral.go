package llm

import (
	"fmt"
	"strings"
	"time"
)

const (
	mistralDefaultBaseURL = "https://api.mistral.ai/v1"
)

// MistralProvider implements the Provider interface for Mistral AI.
// Mistral uses an OpenAI-compatible API format, so this embeds
// CompatibleProvider and overrides only what differs (model info, label).
type MistralProvider struct {
	*CompatibleProvider
}

// NewMistralProvider creates a new Mistral AI provider.
func NewMistralProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("mistral: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = mistralDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &MistralProvider{
		CompatibleProvider: &CompatibleProvider{
			apiKey:  cfg.APIKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  newHTTPClient(timeout),
			label:   "mistral",
		},
	}, nil
}

// ModelInfo returns metadata about the current Mistral model.
func (p *MistralProvider) ModelInfo() ModelMetadata {
	info := ModelMetadata{
		ID:            p.model,
		Provider:      "mistral",
		SupportsTools: true,
	}

	switch {
	case strings.Contains(p.model, "large"):
		info.ContextWindow = 128000
		info.MaxOutput = 4096
		info.CostPer1kIn = 0.002
		info.CostPer1kOut = 0.006
	case strings.Contains(p.model, "medium"):
		info.ContextWindow = 32000
		info.MaxOutput = 4096
		info.CostPer1kIn = 0.0027
		info.CostPer1kOut = 0.0081
	case strings.Contains(p.model, "small"):
		info.ContextWindow = 32000
		info.MaxOutput = 4096
		info.CostPer1kIn = 0.001
		info.CostPer1kOut = 0.003
	case strings.Contains(p.model, "codestral"):
		info.ContextWindow = 32000
		info.MaxOutput = 4096
		info.CostPer1kIn = 0.001
		info.CostPer1kOut = 0.003
	default:
		info.ContextWindow = 32000
		info.MaxOutput = 4096
	}

	return info
}

// RegisterMistral registers the Mistral AI provider factory.
func RegisterMistral(reg *Registry) {
	reg.Register("mistral", func(cfg ProviderConfig) (Provider, error) {
		return NewMistralProvider(cfg)
	})
}
