package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMixtureTool_Name(t *testing.T) {
	tool := &MixtureTool{}
	assert.Equal(t, "mixture", tool.Name())
}

func TestMixtureTool_Description(t *testing.T) {
	tool := &MixtureTool{}
	assert.Contains(t, tool.Description(), "multiple LLM")
}

func TestMixtureTool_InvalidArgs(t *testing.T) {
	tool := &MixtureTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestMixtureTool_DefaultModelsAndMode(t *testing.T) {
	tool := &MixtureTool{}
	args, _ := json.Marshal(mixtureArgs{Prompt: "test prompt"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "compare")
	assert.Contains(t, result.Content, "default")
}

func TestMixtureTool_CustomModels(t *testing.T) {
	tool := &MixtureTool{}
	args, _ := json.Marshal(mixtureArgs{
		Prompt: "test",
		Models: []string{"gpt-4", "claude-3"},
		Mode:   "merge",
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "merge")
	assert.Contains(t, result.Content, "gpt-4")
	assert.Contains(t, result.Content, "claude-3")
}

func TestRegisterMixture(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterMixture(registry)
	_, ok := registry.Get("mixture")
	assert.True(t, ok)
}
