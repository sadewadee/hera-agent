package llm

import (
	"context"
	"fmt"
	"sync"
)

// Provider is the interface all LLM providers must implement.
type Provider interface {
	// Chat sends a non-streaming chat completion request.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream sends a streaming chat completion request.
	// Returns a channel that emits StreamEvents until closed.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)

	// CountTokens estimates the token count for messages.
	// May be approximate for providers without native tokenizers.
	CountTokens(messages []Message) (int, error)

	// ModelInfo returns metadata about the current model.
	ModelInfo() ModelMetadata
}

// ProviderFactory creates a Provider from configuration.
type ProviderFactory func(cfg ProviderConfig) (Provider, error)

// ProviderConfig holds common configuration for providers.
type ProviderConfig struct {
	APIKey       string   `json:"api_key" yaml:"api_key"`
	APIKeys      []string `json:"api_keys,omitempty" yaml:"api_keys,omitempty"`           // optional pool; overrides single APIKey
	PoolStrategy string   `json:"pool_strategy,omitempty" yaml:"pool_strategy,omitempty"` // least_used | round_robin | random
	BaseURL      string   `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Model        string   `json:"model" yaml:"model"`
	OrgID        string   `json:"org_id,omitempty" yaml:"org_id,omitempty"`
	Timeout      int      `json:"timeout,omitempty" yaml:"timeout,omitempty"` // seconds
}

// Registry manages available LLM providers.
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ProviderFactory
	instances map[string]Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ProviderFactory),
		instances: make(map[string]Provider),
	}
}

// Register adds a provider factory to the registry.
func (r *Registry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Create instantiates a provider by name with the given config.
func (r *Registry) Create(name string, cfg ProviderConfig) (Provider, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	provider, err := factory(cfg)
	if err != nil {
		return nil, fmt.Errorf("create provider %s: %w", name, err)
	}

	r.mu.Lock()
	r.instances[name] = provider
	r.mu.Unlock()

	return provider, nil
}

// Get returns an already-instantiated provider.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.instances[name]
	return p, ok
}

// List returns names of all registered provider factories.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}
