package builtin

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvPassthroughTool_Name(t *testing.T) {
	tool := &EnvPassthroughTool{}
	assert.Equal(t, "env_passthrough", tool.Name())
}

func TestEnvPassthroughTool_InvalidArgs(t *testing.T) {
	tool := &EnvPassthroughTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad}`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEnvPassthroughTool_ReadSetVar(t *testing.T) {
	os.Setenv("HERA_TEST_ENV", "test-value-abc")
	defer os.Unsetenv("HERA_TEST_ENV")

	tool := &EnvPassthroughTool{}
	args, _ := json.Marshal(envPassthroughArgs{Names: []string{"HERA_TEST_ENV"}})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "HERA_TEST_ENV=[set")
	assert.Contains(t, result.Content, "14 chars")
}

func TestEnvPassthroughTool_ReadUnsetVar(t *testing.T) {
	os.Unsetenv("HERA_NONEXISTENT_ENV_VAR")
	tool := &EnvPassthroughTool{}
	args, _ := json.Marshal(envPassthroughArgs{Names: []string{"HERA_NONEXISTENT_ENV_VAR"}})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "[not set]")
}

func TestEnvPassthroughTool_MultipleVars(t *testing.T) {
	os.Setenv("HERA_A", "val_a")
	os.Setenv("HERA_B", "val_b")
	defer os.Unsetenv("HERA_A")
	defer os.Unsetenv("HERA_B")

	tool := &EnvPassthroughTool{}
	args, _ := json.Marshal(envPassthroughArgs{Names: []string{"HERA_A", "HERA_B", "HERA_MISSING"}})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "HERA_A=[set")
	assert.Contains(t, result.Content, "HERA_B=[set")
	assert.Contains(t, result.Content, "HERA_MISSING=[not set]")
}
