package llm

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	openrouterDefaultBaseURL = "https://openrouter.ai/api/v1"
)

// OpenRouterProvider implements the Provider interface for OpenRouter.
// OpenRouter is an OpenAI-compatible API that routes to many model providers.
// It requires additional HTTP-Referer and X-Title headers.
type OpenRouterProvider struct {
	*CompatibleProvider
}

// NewOpenRouterProvider creates a new OpenRouter provider.
func NewOpenRouterProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openrouter: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = openrouterDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	apiKey := cfg.APIKey
	referer := "https://github.com/sadewadee/hera"
	title := "Hera AI Agent"

	p := &OpenRouterProvider{
		CompatibleProvider: &CompatibleProvider{
			apiKey:  apiKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  newHTTPClient(timeout),
			label:   "openrouter",
		},
	}

	// Set custom header function that adds OpenRouter-specific headers.
	p.CompatibleProvider.headerSetter = func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("HTTP-Referer", referer)
		req.Header.Set("X-Title", title)
	}

	return p, nil
}

// ModelInfo returns metadata about the current model via OpenRouter.
func (p *OpenRouterProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      "openrouter",
		ContextWindow: 128000,
		MaxOutput:     4096,
		SupportsTools: true,
	}
}

// RegisterOpenRouter registers the OpenRouter provider factory.
func RegisterOpenRouter(reg *Registry) {
	reg.Register("openrouter", func(cfg ProviderConfig) (Provider, error) {
		return NewOpenRouterProvider(cfg)
	})
}
