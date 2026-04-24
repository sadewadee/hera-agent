package memory

import (
	"fmt"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/plugins"
)

// ProviderResult bundles the primary memory.Provider with an optional
// plugins.MemoryProvider sidecar for cloud-augmented memory.
//
// The primary Provider is always a SQLiteProvider — it holds all conversation
// history, facts, and notes used by the core agent.
//
// The Sidecar is non-nil when the user selects an external cloud provider
// (mem0, honcho, etc.) via memory.provider config. The caller wires the
// sidecar's lifecycle hooks (Prefetch, SyncTurn, etc.) into the agent.
// This is additive: the sidecar augments SQLite, it does not replace it.
type ProviderResult struct {
	Primary Provider
	Sidecar plugins.MemoryProvider // nil when provider == "sqlite" or "memory"
}

// NewFromConfig creates a memory Provider based on cfg.
//
// When cfg.Provider is empty, "sqlite", or "memory" the built-in SQLite
// provider is returned and Sidecar is nil.
//
// When cfg.Provider names one of the 8 registered plugin providers, the
// corresponding sidecar is returned alongside the SQLite primary store.
// The sidecar must be initialized (via Sidecar.Initialize) before use.
//
// Returns an error when the named provider is not registered. Call
// RegisterBuiltinProviders before NewFromConfig.
func NewFromConfig(cfg config.MemoryConfig, reg *plugins.Registry, dbPath string) (ProviderResult, error) {
	// Always create the primary SQLite store.
	primary, err := NewSQLiteProvider(dbPath)
	if err != nil {
		return ProviderResult{}, fmt.Errorf("initialize sqlite memory: %w", err)
	}

	providerName := cfg.Provider
	if providerName == "" || providerName == "sqlite" || providerName == "memory" {
		return ProviderResult{Primary: primary}, nil
	}

	// Look up the named plugin provider.
	sidecar := reg.GetMemoryProvider(providerName)
	if sidecar == nil {
		return ProviderResult{}, fmt.Errorf(
			"memory provider %q not registered; valid options: sqlite, memory, "+
				"mem0, hindsight, holographic, honcho, byterover, openviking, retaindb, supermemory",
			providerName,
		)
	}

	if !sidecar.IsAvailable() {
		return ProviderResult{}, fmt.Errorf(
			"memory provider %q is not available (missing required env vars or dependencies); "+
				"check the provider's configuration requirements",
			providerName,
		)
	}

	return ProviderResult{Primary: primary, Sidecar: sidecar}, nil
}
