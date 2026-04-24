package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalTool_Name(t *testing.T) {
	tool := &TerminalTool{}
	assert.Equal(t, "terminal", tool.Name())
}

func TestTerminalTool_Description(t *testing.T) {
	tool := &TerminalTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestTerminalTool_Parameters(t *testing.T) {
	tool := &TerminalTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestTerminalTool_InvalidArgs(t *testing.T) {
	tool := &TerminalTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTerminalTool_SimpleCommand(t *testing.T) {
	tool := &TerminalTool{}
	args, _ := json.Marshal(terminalArgs{Command: "echo test_output"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "test_output")
}

func TestTerminalTool_WithWorkDir(t *testing.T) {
	dir := t.TempDir()
	tool := &TerminalTool{}
	args, _ := json.Marshal(terminalArgs{Command: "pwd", WorkDir: dir})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, dir)
}

func TestTerminalTool_SessionTracking(t *testing.T) {
	tool := &TerminalTool{}
	dir := t.TempDir()
	args, _ := json.Marshal(terminalArgs{Command: "pwd", SessionID: "s1", WorkDir: dir})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, dir)
}

func TestTerminalTool_StderrCapture(t *testing.T) {
	tool := &TerminalTool{}
	args, _ := json.Marshal(terminalArgs{Command: "echo error_msg >&2"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "error_msg")
}
