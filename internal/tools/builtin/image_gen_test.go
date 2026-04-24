package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageGenTool_Name(t *testing.T) {
	tool := &ImageGenTool{}
	assert.Equal(t, "image_generation", tool.Name())
}

func TestImageGenTool_Description(t *testing.T) {
	tool := &ImageGenTool{}
	assert.Contains(t, tool.Description(), "image")
}

func TestImageGenTool_Parameters(t *testing.T) {
	tool := &ImageGenTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestImageGenTool_InvalidArgs(t *testing.T) {
	tool := &ImageGenTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestImageGenTool_EmptyPrompt(t *testing.T) {
	tool := &ImageGenTool{}
	args, _ := json.Marshal(imageGenToolArgs{Prompt: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "prompt is required")
}

func TestImageGenTool_NoAPIKey(t *testing.T) {
	tool := &ImageGenTool{apiKey: ""}
	args, _ := json.Marshal(imageGenToolArgs{Prompt: "a beautiful sunset"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "OPENAI_API_KEY")
}

func TestRegisterImageGen(t *testing.T) {
	registry := tools.NewRegistry()
	t.Setenv("OPENAI_API_KEY", "")
	RegisterImageGen(registry)
	_, ok := registry.Get("image_generation")
	assert.True(t, ok)
}
