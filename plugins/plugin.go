// Package plugins defines the plugin system interfaces for Hera.
//
// Plugins extend Hera's capabilities through well-defined interfaces.
// The two main extension points are:
//   - MemoryProvider: persistent memory backends
//   - ContextEngine: context management strategies
//
// Plugins are discovered by scanning subdirectories under plugins/memory/
// and plugins/context_engine/. Each plugin directory contains a plugin.yaml
// with metadata and a Go implementation that registers via the plugin system.
package plugins

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// PluginMetadata holds the parsed plugin.yaml fields.
type PluginMetadata struct {
	Name                 string   `yaml:"name"`
	Version              string   `yaml:"version"`
	Description          string   `yaml:"description"`
	ExternalDependencies []ExtDep `yaml:"external_dependencies,omitempty"`
	RequiresEnv          []string `yaml:"requires_env,omitempty"`
	Hooks                []string `yaml:"hooks,omitempty"`
}

// ExtDep describes an external binary dependency for a plugin.
type ExtDep struct {
	Name    string `yaml:"name"`
	Install string `yaml:"install"`
	Check   string `yaml:"check"`
}

// ToolSchema describes a tool exposed by a plugin.
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Registry holds all discovered plugins keyed by name.
type Registry struct {
	mu              sync.RWMutex
	memoryProviders map[string]MemoryProvider
	contextEngines  map[string]ContextEngine
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		memoryProviders: make(map[string]MemoryProvider),
		contextEngines:  make(map[string]ContextEngine),
	}
}

// RegisterMemoryProvider adds a memory provider to the registry.
func (r *Registry) RegisterMemoryProvider(p MemoryProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.memoryProviders[p.Name()] = p
	slog.Info("registered memory provider plugin", "name", p.Name())
}

// RegisterContextEngine adds a context engine to the registry.
func (r *Registry) RegisterContextEngine(e ContextEngine) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contextEngines[e.Name()] = e
	slog.Info("registered context engine plugin", "name", e.Name())
}

// GetMemoryProvider returns a memory provider by name, or nil if not found.
func (r *Registry) GetMemoryProvider(name string) MemoryProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.memoryProviders[name]
}

// GetContextEngine returns a context engine by name, or nil if not found.
func (r *Registry) GetContextEngine(name string) ContextEngine {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.contextEngines[name]
}

// ListMemoryProviders returns metadata for all registered memory providers.
func (r *Registry) ListMemoryProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ProviderInfo, 0, len(r.memoryProviders))
	for _, p := range r.memoryProviders {
		result = append(result, ProviderInfo{
			Name:        p.Name(),
			Available:   p.IsAvailable(),
			Description: "", // populated from plugin.yaml at discovery time
		})
	}
	return result
}

// ListContextEngines returns metadata for all registered context engines.
func (r *Registry) ListContextEngines() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ProviderInfo, 0, len(r.contextEngines))
	for _, e := range r.contextEngines {
		result = append(result, ProviderInfo{
			Name:      e.Name(),
			Available: e.IsAvailable(),
		})
	}
	return result
}

// ProviderInfo describes a discovered plugin.
type ProviderInfo struct {
	Name        string
	Description string
	Available   bool
}

// LoadPluginMetadata reads and parses a plugin.yaml file.
func LoadPluginMetadata(dir string) (*PluginMetadata, error) {
	yamlPath := filepath.Join(dir, "plugin.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("reading plugin.yaml: %w", err)
	}
	var meta PluginMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing plugin.yaml: %w", err)
	}
	return &meta, nil
}
