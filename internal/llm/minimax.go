package llm

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	minimaxDefaultBaseURL = "https://api.minimax.chat/v1"
)

// MiniMaxProvider implements the Provider interface for MiniMax.
// MiniMax uses an OpenAI-compatible API format.
type MiniMaxProvider struct {
	*CompatibleProvider
}

// NewMiniMaxProvider creates a new MiniMax provider.
func NewMiniMaxProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("minimax: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = minimaxDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	apiKey := cfg.APIKey

	p := &MiniMaxProvider{
		CompatibleProvider: &CompatibleProvider{
			apiKey:  apiKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  newHTTPClient(timeout),
			label:   "minimax",
		},
	}

	p.CompatibleProvider.headerSetter = func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return p, nil
}

// ModelInfo returns metadata about the current MiniMax model.
func (p *MiniMaxProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      "minimax",
		ContextWindow: 128000,
		MaxOutput:     4096,
		SupportsTools: true,
	}
}

// RegisterMiniMax registers the MiniMax provider factory.
func RegisterMiniMax(reg *Registry) {
	reg.Register("minimax", func(cfg ProviderConfig) (Provider, error) {
		return NewMiniMaxProvider(cfg)
	})
}
