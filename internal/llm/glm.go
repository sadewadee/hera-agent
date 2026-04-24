package llm

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	glmDefaultBaseURL = "https://open.bigmodel.cn/api/paas/v4"
)

// GLMProvider implements the Provider interface for GLM (Zhipu AI).
// GLM uses an OpenAI-compatible API format.
type GLMProvider struct {
	*CompatibleProvider
}

// NewGLMProvider creates a new GLM provider.
func NewGLMProvider(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("glm: api_key is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = glmDefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	apiKey := cfg.APIKey

	p := &GLMProvider{
		CompatibleProvider: &CompatibleProvider{
			apiKey:  apiKey,
			baseURL: baseURL,
			model:   cfg.Model,
			client:  newHTTPClient(timeout),
			label:   "glm",
		},
	}

	p.CompatibleProvider.headerSetter = func(req *http.Request) {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return p, nil
}

// ModelInfo returns metadata about the current GLM model.
func (p *GLMProvider) ModelInfo() ModelMetadata {
	return ModelMetadata{
		ID:            p.model,
		Provider:      "glm",
		ContextWindow: 128000,
		MaxOutput:     4096,
		SupportsTools: true,
	}
}

// RegisterGLM registers the GLM (Zhipu) provider factory.
func RegisterGLM(reg *Registry) {
	reg.Register("glm", func(cfg ProviderConfig) (Provider, error) {
		return NewGLMProvider(cfg)
	})
}
