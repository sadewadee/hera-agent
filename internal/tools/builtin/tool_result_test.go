package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolResultFormatter_Name(t *testing.T) {
	tool := &ToolResultFormatter{}
	assert.Equal(t, "tool_result", tool.Name())
}

func TestToolResultFormatter_InvalidArgs(t *testing.T) {
	tool := &ToolResultFormatter{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestToolResultFormatter_PlainText(t *testing.T) {
	tool := &ToolResultFormatter{}
	args, _ := json.Marshal(toolResultArgs{Content: "hello world"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result.Content)
}

func TestToolResultFormatter_JSONFormat(t *testing.T) {
	tool := &ToolResultFormatter{}
	args, _ := json.Marshal(toolResultArgs{Content: `{"key":"value"}`, Format: "json"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "```json")
}

func TestToolResultFormatter_TableFormat(t *testing.T) {
	tool := &ToolResultFormatter{}
	args, _ := json.Marshal(toolResultArgs{Content: "a\tb\tc", Format: "table"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "a | b | c")
}

func TestToolResultFormatter_MaxLength(t *testing.T) {
	tool := &ToolResultFormatter{}
	args, _ := json.Marshal(toolResultArgs{Content: "hello world long text", MaxLength: 5})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Equal(t, "hello...", result.Content)
}

func TestToolResultFormatter_MarkdownFormat(t *testing.T) {
	tool := &ToolResultFormatter{}
	args, _ := json.Marshal(toolResultArgs{Content: "# Title\nContent", Format: "markdown"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "# Title")
}
