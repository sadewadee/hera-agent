package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOSVTool_Name(t *testing.T) {
	tool := &OSVTool{}
	assert.Equal(t, "osv", tool.Name())
}

func TestOSVTool_Description(t *testing.T) {
	tool := &OSVTool{}
	assert.Contains(t, tool.Description(), "vulnerabilities")
}

func TestOSVTool_InvalidArgs(t *testing.T) {
	tool := &OSVTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid args")
}

func TestOSVTool_Execute(t *testing.T) {
	tool := &OSVTool{}
	args, _ := json.Marshal(osvArgs{Package: "example-pkg", Ecosystem: "npm", Version: "1.0.0"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "example-pkg")
	assert.Contains(t, result.Content, "npm")
}

func TestOSVTool_DefaultEcosystem(t *testing.T) {
	tool := &OSVTool{}
	args, _ := json.Marshal(osvArgs{Package: "mypkg"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result.Content, "Go")
}

func TestRegisterOSV(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterOSV(registry)
	_, ok := registry.Get("osv")
	assert.True(t, ok)
}
