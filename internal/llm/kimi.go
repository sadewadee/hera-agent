package llm

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	kimiDefaultBaseURL = "https://api.moonshot.cn/v1"
)

// KimiProvider implements the Provider interface for Kimi (Moonshot AI).
// Kimi uses an OpenAI-compatible API format.
type KimiProvider struct {
	*CompatibleProvider
}

// NewKimiProvider creates a new Kimi provider.
func NewKimiProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("kimi: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = kimiDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	apiKey := cfg.APIKey

	p := &KimiProvider{
		CompatibleProvider: &CompatibleProvider{
			apiKey:  apiKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  newHTTPClient(timeout),
			label:   "kimi",
		},
	}

	p.CompatibleProvider.headerSetter = func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return p, nil
}

// ModelInfo returns metadata about the current Kimi model.
func (p *KimiProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      "kimi",
		ContextWindow: 128000,
		MaxOutput:     4096,
		SupportsTools: true,
	}
}

// RegisterKimi registers the Kimi (Moonshot) provider factory.
func RegisterKimi(reg *Registry) {
	reg.Register("kimi", func(cfg ProviderConfig) (Provider, error) {
		return NewKimiProvider(cfg)
	})
}
