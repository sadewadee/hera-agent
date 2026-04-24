package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/llm"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/sadewadee/hera/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEngine struct {
	plugins.BaseContextEngine
	toolSchemas []llm.ToolDef
	handleFunc  func(ctx context.Context, name string, args json.RawMessage) (string, error)
}

func (f *fakeEngine) Name() string                                   { return "fake" }
func (f *fakeEngine) IsAvailable() bool                              { return true }
func (f *fakeEngine) Initialize(_ plugins.ContextEngineConfig) error { return nil }
func (f *fakeEngine) UpdateFromResponse(_ llm.Usage)                 {}
func (f *fakeEngine) ShouldCompress(_ int) bool                      { return false }
func (f *fakeEngine) Compress(_ context.Context, msgs []llm.Message, _ int) ([]llm.Message, error) {
	return msgs, nil
}
func (f *fakeEngine) GetToolSchemas() []llm.ToolDef { return f.toolSchemas }
func (f *fakeEngine) HandleToolCall(ctx context.Context, name string, args json.RawMessage) (string, error) {
	if f.handleFunc != nil {
		return f.handleFunc(ctx, name, args)
	}
	return f.BaseContextEngine.HandleToolCall(ctx, name, args)
}

var _ plugins.ContextEngine = (*fakeEngine)(nil)

func TestRegisterEngineTools_NilEngine(t *testing.T) {
	reg := tools.NewRegistry()
	RegisterEngineTools(reg, nil)
	assert.Empty(t, reg.List())
}

func TestRegisterEngineTools_NoSchemas(t *testing.T) {
	reg := tools.NewRegistry()
	eng := &fakeEngine{}
	RegisterEngineTools(reg, eng)
	assert.Empty(t, reg.List())
}

func TestRegisterEngineTools_RegistersAndExecutes(t *testing.T) {
	params := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	eng := &fakeEngine{
		toolSchemas: []llm.ToolDef{
			{Name: "ctx_search", Description: "Search context graph", Parameters: params},
			{Name: "ctx_expand", Description: "Expand context node", Parameters: params},
		},
		handleFunc: func(_ context.Context, name string, args json.RawMessage) (string, error) {
			return "handled:" + name, nil
		},
	}

	reg := tools.NewRegistry()
	RegisterEngineTools(reg, eng)

	list := reg.List()
	require.Len(t, list, 2)

	names := make([]string, len(list))
	for i, t := range list {
		names[i] = t.Name()
	}
	assert.Contains(t, names, "ctx_search")
	assert.Contains(t, names, "ctx_expand")

	tool, ok := reg.Get("ctx_search")
	require.True(t, ok)
	assert.Equal(t, "Search context graph", tool.Description())

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test"}`))
	require.NoError(t, err)
	assert.Equal(t, "handled:ctx_search", result.Content)
	assert.False(t, result.IsError)
}
