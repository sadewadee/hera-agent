package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeExecTool_Name(t *testing.T) {
	tool := &CodeExecTool{}
	assert.Equal(t, "code_exec", tool.Name())
}

func TestCodeExecTool_Description(t *testing.T) {
	tool := &CodeExecTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestCodeExecTool_Parameters(t *testing.T) {
	tool := &CodeExecTool{}
	params := tool.Parameters()
	assert.NotNil(t, params)
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestCodeExecTool_InvalidArgs(t *testing.T) {
	tool := &CodeExecTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{invalid`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid args")
}

func TestCodeExecTool_UnsupportedLanguage(t *testing.T) {
	tool := &CodeExecTool{}
	args := json.RawMessage(`{"language":"ruby","code":"puts 'hello'"}`)
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unsupported language")
}

func TestCodeExecTool_ExecuteBash(t *testing.T) {
	tool := &CodeExecTool{}
	args := json.RawMessage(`{"language":"bash","code":"echo hello world"}`)
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "hello world")
}

func TestCodeExecTool_ExecuteBash_Error(t *testing.T) {
	tool := &CodeExecTool{}
	args := json.RawMessage(`{"language":"bash","code":"exit 1"}`)
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Error")
}
