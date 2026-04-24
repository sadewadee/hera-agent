package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTodoTool_Name(t *testing.T) {
	tool := &TodoTool{}
	assert.Equal(t, "todo", tool.Name())
}

func TestTodoTool_ListEmpty(t *testing.T) {
	tool := &TodoTool{}
	args, _ := json.Marshal(todoArgs{Action: "list"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "No todos")
}

func TestTodoTool_Add(t *testing.T) {
	tool := &TodoTool{}
	args, _ := json.Marshal(todoArgs{Action: "add", Text: "Write tests"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Added todo #1")
	assert.Contains(t, result.Content, "Write tests")
}

func TestTodoTool_Add_EmptyText(t *testing.T) {
	tool := &TodoTool{}
	args, _ := json.Marshal(todoArgs{Action: "add", Text: ""})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "text required")
}

func TestTodoTool_ListAfterAdd(t *testing.T) {
	tool := &TodoTool{}
	addArgs, _ := json.Marshal(todoArgs{Action: "add", Text: "Item 1"})
	tool.Execute(context.Background(), addArgs)
	addArgs, _ = json.Marshal(todoArgs{Action: "add", Text: "Item 2"})
	tool.Execute(context.Background(), addArgs)

	listArgs, _ := json.Marshal(todoArgs{Action: "list"})
	result, err := tool.Execute(context.Background(), listArgs)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Item 1")
	assert.Contains(t, result.Content, "Item 2")
	assert.Contains(t, result.Content, "[ ]")
}

func TestTodoTool_Complete(t *testing.T) {
	tool := &TodoTool{}
	addArgs, _ := json.Marshal(todoArgs{Action: "add", Text: "Task"})
	tool.Execute(context.Background(), addArgs)

	completeArgs, _ := json.Marshal(todoArgs{Action: "complete", ID: 1})
	result, err := tool.Execute(context.Background(), completeArgs)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Completed #1")

	// Verify it shows as done
	listArgs, _ := json.Marshal(todoArgs{Action: "list"})
	result, _ = tool.Execute(context.Background(), listArgs)
	assert.Contains(t, result.Content, "[x]")
}

func TestTodoTool_Complete_NotFound(t *testing.T) {
	tool := &TodoTool{}
	args, _ := json.Marshal(todoArgs{Action: "complete", ID: 99})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "not found")
}

func TestTodoTool_Remove(t *testing.T) {
	tool := &TodoTool{}
	addArgs, _ := json.Marshal(todoArgs{Action: "add", Text: "Task"})
	tool.Execute(context.Background(), addArgs)

	removeArgs, _ := json.Marshal(todoArgs{Action: "remove", ID: 1})
	result, err := tool.Execute(context.Background(), removeArgs)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Removed #1")
}

func TestTodoTool_Remove_NotFound(t *testing.T) {
	tool := &TodoTool{}
	args, _ := json.Marshal(todoArgs{Action: "remove", ID: 99})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTodoTool_UnknownAction(t *testing.T) {
	tool := &TodoTool{}
	args, _ := json.Marshal(todoArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestTodoTool_InvalidJSON(t *testing.T) {
	tool := &TodoTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
