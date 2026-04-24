package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTirithTool_Name(t *testing.T) {
	tool := &TirithTool{}
	assert.Equal(t, "tirith", tool.Name())
}

func TestTirithTool_Description(t *testing.T) {
	tool := &TirithTool{}
	assert.Contains(t, tool.Description(), "policy")
}

func TestTirithTool_InvalidArgs(t *testing.T) {
	tool := &TirithTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTirithTool_Execute(t *testing.T) {
	tool := &TirithTool{}
	args, _ := json.Marshal(tirithArgs{Policy: "no-root", Input: `{"user": "admin"}`})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "no-root")
	assert.Contains(t, result.Content, "PASS")
}

func TestRegisterTirith(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterTirith(registry)
	_, ok := registry.Get("tirith")
	assert.True(t, ok)
}
