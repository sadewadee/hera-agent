package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugTool_Name(t *testing.T) {
	tool := &DebugTool{}
	assert.Equal(t, "debug", tool.Name())
}

func TestDebugTool_Execute_All(t *testing.T) {
	tool := &DebugTool{}
	args, _ := json.Marshal(debugArgs{Section: "all"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Go Version")
	assert.Contains(t, result.Content, "Goroutines")
	assert.Contains(t, result.Content, "Heap Alloc")
}

func TestDebugTool_Execute_Default(t *testing.T) {
	tool := &DebugTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Go Version")
}

func TestDebugTool_Execute_Runtime(t *testing.T) {
	tool := &DebugTool{}
	args, _ := json.Marshal(debugArgs{Section: "runtime"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)
}
