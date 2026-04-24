package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeCodeTool_Name(t *testing.T) {
	tool := &ClaudeCodeTool{}
	assert.Equal(t, "claude_code", tool.Name())
}

func TestClaudeCodeTool_Description(t *testing.T) {
	tool := &ClaudeCodeTool{}
	assert.Contains(t, tool.Description(), "Claude Code")
}

func TestClaudeCodeTool_Parameters(t *testing.T) {
	tool := &ClaudeCodeTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestClaudeCodeTool_InvalidArgs(t *testing.T) {
	tool := &ClaudeCodeTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestClaudeCodeTool_EmptyPrompt(t *testing.T) {
	tool := &ClaudeCodeTool{}
	args, _ := json.Marshal(claudeCodeArgs{Prompt: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "prompt is required")
}

func TestRegisterClaudeCode(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterClaudeCode(registry)
	_, ok := registry.Get("claude_code")
	assert.True(t, ok)
}
