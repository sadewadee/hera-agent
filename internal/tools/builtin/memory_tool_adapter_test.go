package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeMemoryProvider struct {
	schemas    []plugins.ToolSchema
	handleFunc func(name string, args map[string]interface{}) (string, error)
}

func (f *fakeMemoryProvider) Name() string                                  { return "fake-mem" }
func (f *fakeMemoryProvider) IsAvailable() bool                             { return true }
func (f *fakeMemoryProvider) Initialize(string) error                       { return nil }
func (f *fakeMemoryProvider) SystemPromptBlock() string                     { return "" }
func (f *fakeMemoryProvider) Prefetch(string, string) string                { return "" }
func (f *fakeMemoryProvider) SyncTurn(string, string, string)               {}
func (f *fakeMemoryProvider) OnMemoryWrite(string, string, string)          {}
func (f *fakeMemoryProvider) OnPreCompress([]map[string]interface{}) string { return "" }
func (f *fakeMemoryProvider) OnSessionEnd([]map[string]interface{})         {}
func (f *fakeMemoryProvider) GetToolSchemas() []plugins.ToolSchema          { return f.schemas }
func (f *fakeMemoryProvider) HandleToolCall(name string, args map[string]interface{}) (string, error) {
	if f.handleFunc != nil {
		return f.handleFunc(name, args)
	}
	return "", nil
}
func (f *fakeMemoryProvider) GetConfigSchema() []plugins.ConfigField { return nil }
func (f *fakeMemoryProvider) Shutdown()                              {}

var _ plugins.MemoryProvider = (*fakeMemoryProvider)(nil)

func TestRegisterMemoryProviderTools_NilProvider(t *testing.T) {
	reg := tools.NewRegistry()
	RegisterMemoryProviderTools(reg, nil)
	assert.Empty(t, reg.List())
}

func TestRegisterMemoryProviderTools_NoSchemas(t *testing.T) {
	reg := tools.NewRegistry()
	RegisterMemoryProviderTools(reg, &fakeMemoryProvider{})
	assert.Empty(t, reg.List())
}

func TestRegisterMemoryProviderTools_RegistersAndExecutes(t *testing.T) {
	provider := &fakeMemoryProvider{
		schemas: []plugins.ToolSchema{
			{
				Name:        "hindsight_recall",
				Description: "Search knowledge graph",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{"type": "string"},
					},
					"required": []string{"query"},
				},
			},
			{
				Name:        "hindsight_memorize",
				Description: "Store fact",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"content": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
		handleFunc: func(name string, args map[string]interface{}) (string, error) {
			return "called:" + name + ":" + args["query"].(string), nil
		},
	}

	reg := tools.NewRegistry()
	RegisterMemoryProviderTools(reg, provider)

	list := reg.List()
	require.Len(t, list, 2)

	tool, ok := reg.Get("hindsight_recall")
	require.True(t, ok)
	assert.Equal(t, "Search knowledge graph", tool.Description())

	// Verify parameters were JSON-marshaled correctly.
	var params map[string]interface{}
	require.NoError(t, json.Unmarshal(tool.Parameters(), &params))
	assert.Equal(t, "object", params["type"])

	// Execute with JSON args, confirm provider receives map form.
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"ping"}`))
	require.NoError(t, err)
	assert.Equal(t, "called:hindsight_recall:ping", result.Content)
	assert.False(t, result.IsError)
}

func TestRegisterMemoryProviderTools_ExecuteSurfacesHandleError(t *testing.T) {
	provider := &fakeMemoryProvider{
		schemas: []plugins.ToolSchema{
			{Name: "failing_tool", Description: "Always fails", Parameters: map[string]interface{}{}},
		},
		handleFunc: func(string, map[string]interface{}) (string, error) {
			return "", errors.New("provider exploded")
		},
	}

	reg := tools.NewRegistry()
	RegisterMemoryProviderTools(reg, provider)

	tool, ok := reg.Get("failing_tool")
	require.True(t, ok)

	result, err := tool.Execute(context.Background(), nil)
	require.NoError(t, err) // Execute returns (Result{IsError: true}, nil), not a Go error.
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "provider exploded")
}

func TestRegisterMemoryProviderTools_ExecuteHandlesBadJSON(t *testing.T) {
	provider := &fakeMemoryProvider{
		schemas: []plugins.ToolSchema{
			{Name: "x", Description: "", Parameters: map[string]interface{}{}},
		},
	}

	reg := tools.NewRegistry()
	RegisterMemoryProviderTools(reg, provider)

	tool, _ := reg.Get("x")
	result, err := tool.Execute(context.Background(), json.RawMessage(`{not-json`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid arguments")
}
