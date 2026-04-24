package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestK8sTool_Name(t *testing.T) {
	tool := &K8sTool{}
	assert.Equal(t, "kubernetes", tool.Name())
}

func TestK8sTool_Description(t *testing.T) {
	tool := &K8sTool{}
	assert.Contains(t, tool.Description(), "Kubernetes")
}

func TestK8sTool_Parameters(t *testing.T) {
	tool := &K8sTool{}
	params := tool.Parameters()
	var schema map[string]interface{}
	require.NoError(t, json.Unmarshal(params, &schema))
	assert.Equal(t, "object", schema["type"])
}

func TestK8sTool_InvalidArgs(t *testing.T) {
	tool := &K8sTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestK8sTool_GetRequiresResource(t *testing.T) {
	tool := &K8sTool{}
	args, _ := json.Marshal(k8sToolArgs{Action: "get"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "resource type is required")
}

func TestK8sTool_DescribeRequiresResourceAndName(t *testing.T) {
	tool := &K8sTool{}
	args, _ := json.Marshal(k8sToolArgs{Action: "describe", Resource: "pod"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "resource and name are required")
}

func TestK8sTool_LogsRequiresName(t *testing.T) {
	tool := &K8sTool{}
	args, _ := json.Marshal(k8sToolArgs{Action: "logs"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "pod name is required")
}

func TestK8sTool_UnknownAction(t *testing.T) {
	tool := &K8sTool{}
	args, _ := json.Marshal(k8sToolArgs{Action: "invalid"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestRegisterK8s(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterK8s(registry)
	_, ok := registry.Get("kubernetes")
	assert.True(t, ok)
}
