package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalTool_Name(t *testing.T) {
	tool := &ApprovalTool{}
	assert.Equal(t, "approval", tool.Name())
}

func TestApprovalTool_Description(t *testing.T) {
	tool := &ApprovalTool{}
	assert.NotEmpty(t, tool.Description())
}

func TestApprovalTool_InvalidArgs(t *testing.T) {
	tool := &ApprovalTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestApprovalTool_Execute_DefaultRisk(t *testing.T) {
	tool := &ApprovalTool{}
	args, _ := json.Marshal(approvalArgs{Action: "delete production db", Description: "Will drop all tables"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "APPROVAL REQUIRED")
	assert.Contains(t, result.Content, "medium")
	assert.Contains(t, result.Content, "delete production db")
}

func TestApprovalTool_Execute_HighRisk(t *testing.T) {
	tool := &ApprovalTool{}
	args, _ := json.Marshal(approvalArgs{Action: "deploy", Description: "Deploy to prod", Risk: "high"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "high")
}

func TestApprovalTool_Execute_CriticalRisk(t *testing.T) {
	tool := &ApprovalTool{}
	args, _ := json.Marshal(approvalArgs{Action: "delete", Description: "Irreversible", Risk: "critical"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "critical")
}
