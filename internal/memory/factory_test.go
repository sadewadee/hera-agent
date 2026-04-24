package memory_test

import (
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/config"
	"github.com/sadewadee/hera/internal/memory"
	"github.com/sadewadee/hera/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewFromConfig_SQLiteDefault verifies that an empty provider name
// returns a valid SQLite-backed ProviderResult with no sidecar.
func TestNewFromConfig_SQLiteDefault(t *testing.T) {
	reg := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(reg)

	dbPath := filepath.Join(t.TempDir(), "test.db")
	cfg := config.MemoryConfig{Provider: ""}

	result, err := memory.NewFromConfig(cfg, reg, dbPath)
	require.NoError(t, err)
	assert.NotNil(t, result.Primary)
	assert.Nil(t, result.Sidecar, "no sidecar for default sqlite provider")
	require.NoError(t, result.Primary.Close())
}

// TestNewFromConfig_ExplicitSQLite verifies that provider: "sqlite" returns
// the built-in store with no sidecar.
func TestNewFromConfig_ExplicitSQLite(t *testing.T) {
	reg := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(reg)

	dbPath := filepath.Join(t.TempDir(), "test.db")
	cfg := config.MemoryConfig{Provider: "sqlite"}

	result, err := memory.NewFromConfig(cfg, reg, dbPath)
	require.NoError(t, err)
	assert.NotNil(t, result.Primary)
	assert.Nil(t, result.Sidecar)
	require.NoError(t, result.Primary.Close())
}

// TestNewFromConfig_UnknownProvider verifies that an unregistered provider
// name returns an error.
func TestNewFromConfig_UnknownProvider(t *testing.T) {
	reg := plugins.NewRegistry()
	// Do NOT call RegisterBuiltinProviders — registry is empty.

	dbPath := filepath.Join(t.TempDir(), "test.db")
	cfg := config.MemoryConfig{Provider: "nonexistent"}

	_, err := memory.NewFromConfig(cfg, reg, dbPath)
	require.Error(t, err, "unknown provider should return error")
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestNewFromConfig_HolographicProviderRegistered verifies that the
// holographic provider (local, no external deps) is registered and can
// be selected without error. Holographic uses the same SQLite driver as
// the main app — no external API key required.
func TestNewFromConfig_HolographicProviderRegistered(t *testing.T) {
	reg := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(reg)

	// Holographic must be discoverable in the registry.
	p := reg.GetMemoryProvider("holographic")
	require.NotNil(t, p, "holographic provider must be registered")
	assert.Equal(t, "holographic", p.Name())
}

// TestRegisterBuiltinProviders_AllEight verifies all 8 providers are registered.
func TestRegisterBuiltinProviders_AllEight(t *testing.T) {
	reg := plugins.NewRegistry()
	memory.RegisterBuiltinProviders(reg)

	expected := []string{
		"mem0", "hindsight", "holographic", "honcho",
		"byterover", "openviking", "retaindb", "supermemory",
	}
	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			p := reg.GetMemoryProvider(name)
			assert.NotNil(t, p, "provider %q must be registered", name)
			assert.Equal(t, name, p.Name())
		})
	}
}
