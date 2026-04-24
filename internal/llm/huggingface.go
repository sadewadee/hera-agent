package llm

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	hfDefaultBaseURL = "https://api-inference.huggingface.co/v1"
)

// HuggingFaceProvider implements the Provider interface for HuggingFace Inference API.
// HuggingFace TGI exposes an OpenAI-compatible endpoint.
type HuggingFaceProvider struct {
	*CompatibleProvider
}

// NewHuggingFaceProvider creates a new HuggingFace provider.
func NewHuggingFaceProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("huggingface: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = hfDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	apiKey := cfg.APIKey

	p := &HuggingFaceProvider{
		CompatibleProvider: &CompatibleProvider{
			apiKey:  apiKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  newHTTPClient(timeout),
			label:   "huggingface",
		},
	}

	p.CompatibleProvider.headerSetter = func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return p, nil
}

// ModelInfo returns metadata about the current HuggingFace model.
func (p *HuggingFaceProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      "huggingface",
		ContextWindow: 32000,
		MaxOutput:     4096,
		SupportsTools: true,
	}
}

// RegisterHuggingFace registers the HuggingFace provider factory.
func RegisterHuggingFace(reg *Registry) {
	reg.Register("huggingface", func(cfg ProviderConfig) (Provider, error) {
		return NewHuggingFaceProvider(cfg)
	})
}
