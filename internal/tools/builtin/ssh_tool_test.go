package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHTool_Name(t *testing.T) {
	tool := &SSHTool{}
	assert.Equal(t, "ssh", tool.Name())
}

func TestSSHTool_Description(t *testing.T) {
	tool := &SSHTool{}
	assert.Contains(t, tool.Description(), "SSH")
}

func TestSSHTool_Parameters(t *testing.T) {
	tool := &SSHTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestSSHTool_InvalidArgs(t *testing.T) {
	tool := &SSHTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestRegisterSSH(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterSSH(registry)
	_, ok := registry.Get("ssh")
	assert.True(t, ok)
}
