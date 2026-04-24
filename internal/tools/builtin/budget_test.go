package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetTool_Name(t *testing.T) {
	tool := &BudgetTool{}
	assert.Equal(t, "budget", tool.Name())
}

func TestBudgetTool_Description(t *testing.T) {
	tool := &BudgetTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestBudgetTool_InvalidArgs(t *testing.T) {
	tool := &BudgetTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBudgetTool_Status(t *testing.T) {
	tool := &BudgetTool{}
	args, _ := json.Marshal(budgetArgs{Action: "status"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Budget:")
}

func TestBudgetTool_SetLimit(t *testing.T) {
	tool := &BudgetTool{}
	args, _ := json.Marshal(budgetArgs{Action: "set_limit", MaxTokens: 100000})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "100000")
}

func TestBudgetTool_AddUsage(t *testing.T) {
	tool := &BudgetTool{}
	args, _ := json.Marshal(budgetArgs{Action: "add_usage", Tokens: 500})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Added 500 tokens")

	// Check status reflects usage
	statusArgs, _ := json.Marshal(budgetArgs{Action: "status"})
	result, _ = tool.Execute(context.Background(), statusArgs)
	assert.Contains(t, result.Content, "500")
}

func TestBudgetTool_Reset(t *testing.T) {
	tool := &BudgetTool{}
	addArgs, _ := json.Marshal(budgetArgs{Action: "add_usage", Tokens: 1000})
	tool.Execute(context.Background(), addArgs)

	resetArgs, _ := json.Marshal(budgetArgs{Action: "reset"})
	result, err := tool.Execute(context.Background(), resetArgs)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Budget reset")
}

func TestBudgetTool_UnknownAction(t *testing.T) {
	tool := &BudgetTool{}
	args, _ := json.Marshal(budgetArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
