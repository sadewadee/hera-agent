package llm

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	nousDefaultBaseURL = "https://inference-api.nousresearch.com/v1"
)

// NousProvider implements the Provider interface for Nous Research Portal.
// Nous uses an OpenAI-compatible API.
type NousProvider struct {
	*CompatibleProvider
}

// NewNousProvider creates a new Nous Research provider.
func NewNousProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("nous: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = nousDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	apiKey := cfg.APIKey

	p := &NousProvider{
		CompatibleProvider: &CompatibleProvider{
			apiKey:  apiKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  newHTTPClient(timeout),
			label:   "nous",
		},
	}

	p.CompatibleProvider.headerSetter = func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return p, nil
}

// ModelInfo returns metadata about the current Nous model.
func (p *NousProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      "nous",
		ContextWindow: 128000,
		MaxOutput:     4096,
		SupportsTools: true,
	}
}

// RegisterNous registers the Nous Research provider factory.
func RegisterNous(reg *Registry) {
	reg.Register("nous", func(cfg ProviderConfig) (Provider, error) {
		return NewNousProvider(cfg)
	})
}
