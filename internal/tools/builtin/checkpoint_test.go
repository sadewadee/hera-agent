package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointTool_Name(t *testing.T) {
	tool := &CheckpointTool{}
	assert.Equal(t, "checkpoint", tool.Name())
}

func TestCheckpointTool_Description(t *testing.T) {
	tool := &CheckpointTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestCheckpointTool_InvalidArgs(t *testing.T) {
	tool := &CheckpointTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestCheckpointTool_Save(t *testing.T) {
	tool := &CheckpointTool{}
	args, _ := json.Marshal(checkpointArgs{Label: "before-refactor"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "before-refactor")
	assert.Contains(t, result.Content, "saved at")
}

func TestCheckpointTool_SaveWithData(t *testing.T) {
	tool := &CheckpointTool{}
	args, _ := json.Marshal(checkpointArgs{Label: "snapshot", Data: "state=active"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "snapshot")
}
