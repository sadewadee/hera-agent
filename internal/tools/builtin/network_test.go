package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkTool_Name(t *testing.T) {
	tool := &NetworkTool{}
	assert.Equal(t, "network", tool.Name())
}

func TestNetworkTool_Description(t *testing.T) {
	tool := &NetworkTool{}
	assert.Contains(t, tool.Description(), "Network")
}

func TestNetworkTool_InvalidArgs(t *testing.T) {
	tool := &NetworkTool{}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{bad`))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNetworkTool_DNSLookup(t *testing.T) {
	tool := &NetworkTool{}
	args, _ := json.Marshal(networkArgs{Action: "dns", Host: "localhost", Timeout: 2})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	// localhost should resolve
	assert.Contains(t, result.Content, "DNS lookup")
}

func TestNetworkTool_PortCheckInvalidPort(t *testing.T) {
	tool := &NetworkTool{}
	args, _ := json.Marshal(networkArgs{Action: "port_check", Host: "localhost", Port: 0})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "invalid port")
}

func TestNetworkTool_PortCheckHighPort(t *testing.T) {
	tool := &NetworkTool{}
	args, _ := json.Marshal(networkArgs{Action: "port_check", Host: "localhost", Port: 99999})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNetworkTool_UnknownAction(t *testing.T) {
	tool := &NetworkTool{}
	args, _ := json.Marshal(networkArgs{Action: "invalid", Host: "localhost"})
	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "unknown action")
}

func TestRegisterNetwork(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterNetwork(registry)
	_, ok := registry.Get("network")
	assert.True(t, ok)
}
