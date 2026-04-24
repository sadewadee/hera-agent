package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterruptTool_Name(t *testing.T) {
	tool := &InterruptTool{}
	assert.Equal(t, "interrupt", tool.Name())
}

func TestInterruptTool_InvalidArgs(t *testing.T) {
	tool := &InterruptTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestInterruptTool_ImmediateInterrupt(t *testing.T) {
	tool := &InterruptTool{}
	args, _ := json.Marshal(interruptArgs{Reason: "user cancelled"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "INTERRUPT")
	assert.Contains(t, result.Content, "immediate")
	assert.Contains(t, result.Content, "user cancelled")
}

func TestInterruptTool_GracefulInterrupt(t *testing.T) {
	tool := &InterruptTool{}
	args, _ := json.Marshal(interruptArgs{Reason: "timeout", Graceful: true})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "graceful")
	assert.Contains(t, result.Content, "timeout")
}
