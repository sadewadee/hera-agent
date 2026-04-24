package plugins

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	require.NotNil(t, reg)
	assert.NotNil(t, reg.memoryProviders)
	assert.NotNil(t, reg.contextEngines)
}

type fakeMemoryProvider struct {
	name      string
	available bool
}

func (f *fakeMemoryProvider) Name() string                                  { return f.name }
func (f *fakeMemoryProvider) IsAvailable() bool                             { return f.available }
func (f *fakeMemoryProvider) Initialize(string) error                       { return nil }
func (f *fakeMemoryProvider) SystemPromptBlock() string                     { return "" }
func (f *fakeMemoryProvider) Prefetch(string, string) string                { return "" }
func (f *fakeMemoryProvider) SyncTurn(string, string, string)               {}
func (f *fakeMemoryProvider) OnMemoryWrite(string, string, string)          {}
func (f *fakeMemoryProvider) OnPreCompress([]map[string]interface{}) string { return "" }
func (f *fakeMemoryProvider) OnSessionEnd([]map[string]interface{})         {}
func (f *fakeMemoryProvider) GetToolSchemas() []ToolSchema                  { return nil }
func (f *fakeMemoryProvider) HandleToolCall(string, map[string]interface{}) (string, error) {
	return "", nil
}
func (f *fakeMemoryProvider) GetConfigSchema() []ConfigField { return nil }
func (f *fakeMemoryProvider) Shutdown()                      {}

type fakeContextEngine struct {
	BaseContextEngine
	name      string
	available bool
}

func (f *fakeContextEngine) Name() string      { return f.name }
func (f *fakeContextEngine) IsAvailable() bool { return f.available }
func (f *fakeContextEngine) Initialize(cfg ContextEngineConfig) error {
	f.InitBase(cfg)
	return nil
}
func (f *fakeContextEngine) UpdateFromResponse(usage llm.Usage) { f.RecordUsage(usage) }
func (f *fakeContextEngine) ShouldCompress(promptTokens int) bool {
	return promptTokens > f.ThresholdTokens()
}
func (f *fakeContextEngine) Compress(_ context.Context, msgs []llm.Message, _ int) ([]llm.Message, error) {
	return msgs, nil
}

// Implement optional overrides explicitly to satisfy the interface via
// BaseContextEngine embedding — the compiler verifies all 15 methods present.
var _ ContextEngine = (*fakeContextEngine)(nil)

// HandleToolCall delegates to BaseContextEngine default.
func (f *fakeContextEngine) HandleToolCall(ctx context.Context, name string, args json.RawMessage) (string, error) {
	return f.BaseContextEngine.HandleToolCall(ctx, name, args)
}

func TestRegistry_RegisterMemoryProvider(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterMemoryProvider(&fakeMemoryProvider{name: "test-mem", available: true})
	got := reg.GetMemoryProvider("test-mem")
	require.NotNil(t, got)
	assert.Equal(t, "test-mem", got.Name())
}

func TestRegistry_GetMemoryProvider_NotFound(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetMemoryProvider("missing"))
}

func TestRegistry_RegisterContextEngine(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterContextEngine(&fakeContextEngine{name: "test-engine", available: true})
	got := reg.GetContextEngine("test-engine")
	require.NotNil(t, got)
	assert.Equal(t, "test-engine", got.Name())
}

func TestRegistry_GetContextEngine_NotFound(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetContextEngine("missing"))
}

func TestRegistry_ListMemoryProviders(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterMemoryProvider(&fakeMemoryProvider{name: "p1", available: true})
	reg.RegisterMemoryProvider(&fakeMemoryProvider{name: "p2", available: false})
	list := reg.ListMemoryProviders()
	assert.Len(t, list, 2)
}

func TestRegistry_ListContextEngines(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterContextEngine(&fakeContextEngine{name: "e1", available: true})
	list := reg.ListContextEngines()
	assert.Len(t, list, 1)
	assert.Equal(t, "e1", list[0].Name)
	assert.True(t, list[0].Available)
}

func TestRegistry_ListEmpty(t *testing.T) {
	reg := NewRegistry()
	assert.Empty(t, reg.ListMemoryProviders())
	assert.Empty(t, reg.ListContextEngines())
}

func TestLoadPluginMetadata(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `name: test-plugin
version: 1.0.0
description: A test plugin
requires_env:
  - TEST_KEY
hooks:
  - on_start
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(yamlContent), 0o644))

	meta, err := LoadPluginMetadata(dir)
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", meta.Name)
	assert.Equal(t, "1.0.0", meta.Version)
	assert.Equal(t, "A test plugin", meta.Description)
	assert.Equal(t, []string{"TEST_KEY"}, meta.RequiresEnv)
	assert.Equal(t, []string{"on_start"}, meta.Hooks)
}

func TestLoadPluginMetadata_NotFound(t *testing.T) {
	_, err := LoadPluginMetadata("/nonexistent")
	assert.Error(t, err)
}

func TestLoadPluginMetadata_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte("invalid: [yaml: bad"), 0o644))
	_, err := LoadPluginMetadata(dir)
	assert.Error(t, err)
}

func TestPluginMetadata_ExternalDeps(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `name: dep-plugin
version: 0.1.0
description: Has deps
external_dependencies:
  - name: redis
    install: brew install redis
    check: redis-cli --version
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(yamlContent), 0o644))

	meta, err := LoadPluginMetadata(dir)
	require.NoError(t, err)
	require.Len(t, meta.ExternalDependencies, 1)
	assert.Equal(t, "redis", meta.ExternalDependencies[0].Name)
	assert.Equal(t, "brew install redis", meta.ExternalDependencies[0].Install)
}

func TestProviderInfo(t *testing.T) {
	info := ProviderInfo{
		Name:        "test",
		Description: "Test provider",
		Available:   true,
	}
	assert.Equal(t, "test", info.Name)
	assert.True(t, info.Available)
}

func TestConfigField(t *testing.T) {
	cf := ConfigField{
		Key:         "api_key",
		Description: "The API key",
		Secret:      true,
		EnvVar:      "MY_API_KEY",
	}
	assert.True(t, cf.Secret)
	assert.Equal(t, "MY_API_KEY", cf.EnvVar)
}
