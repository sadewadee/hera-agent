package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClarifyTool_Name(t *testing.T) {
	tool := &ClarifyTool{}
	assert.Equal(t, "clarify", tool.Name())
}

func TestClarifyTool_Description(t *testing.T) {
	tool := &ClarifyTool{}
	assert.Contains(t, tool.Description(), "clarifying")
}

func TestClarifyTool_InvalidArgs(t *testing.T) {
	tool := &ClarifyTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestClarifyTool_QuestionOnly(t *testing.T) {
	tool := &ClarifyTool{}
	args, _ := json.Marshal(clarifyArgs{Question: "Which format do you prefer?"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "CLARIFICATION NEEDED")
	assert.Contains(t, result.Content, "Which format")
}

func TestClarifyTool_WithOptions(t *testing.T) {
	tool := &ClarifyTool{}
	args, _ := json.Marshal(clarifyArgs{
		Question: "Pick a color",
		Options:  []string{"red", "blue", "green"},
	})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Pick a color")
	assert.Contains(t, result.Content, "Options")
	assert.Contains(t, result.Content, "red")
}

func TestRegisterClarify(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterClarify(registry)
	_, ok := registry.Get("clarify")
	assert.True(t, ok)
}
