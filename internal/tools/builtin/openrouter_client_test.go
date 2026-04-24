package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRouterTool_Name(t *testing.T) {
	tool := &OpenRouterTool{}
	assert.Equal(t, "openrouter_query", tool.Name())
}

func TestOpenRouterTool_Description(t *testing.T) {
	tool := &OpenRouterTool{}
	assert.Contains(t, tool.Description(), "OpenRouter")
}

func TestOpenRouterTool_InvalidArgs(t *testing.T) {
	tool := &OpenRouterTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestOpenRouterTool_NoAPIKey(t *testing.T) {
	tool := &OpenRouterTool{apiKey: ""}
	t.Setenv("OPENROUTER_API_KEY", "")
	args, _ := json.Marshal(openrouterArgs{Model: "test", Prompt: "hello"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "OPENROUTER_API_KEY")
}

func TestOpenRouterTool_WithAPIKey(t *testing.T) {
	tool := &OpenRouterTool{apiKey: "test-key"}
	args, _ := json.Marshal(openrouterArgs{Model: "anthropic/claude-3", Prompt: "hello world"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "anthropic/claude-3")
}

func TestRegisterOpenRouterClient(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterOpenRouterClient(registry)
	_, ok := registry.Get("openrouter_query")
	assert.True(t, ok)
}
