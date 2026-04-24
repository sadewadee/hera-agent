package builtin

import (
	"encoding/json"
	"testing"

	"github.com/sadewadee/hera/internal/mcp"
	"github.com/sadewadee/hera/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPProxyTool_Name(t *testing.T) {
	tool := &MCPProxyTool{
		serverName: "test-server",
		toolDef:    mcp.MCPToolDef{Name: "my_tool"},
	}
	assert.Equal(t, "mcp__test-server__my_tool", tool.Name())
}

func TestMCPProxyTool_Description(t *testing.T) {
	tool := &MCPProxyTool{
		serverName: "srv",
		toolDef:    mcp.MCPToolDef{Description: "Does something useful"},
	}
	assert.Contains(t, tool.Description(), "MCP:srv")
	assert.Contains(t, tool.Description(), "Does something useful")
}

func TestMCPProxyTool_Parameters(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	tool := &MCPProxyTool{
		toolDef: mcp.MCPToolDef{InputSchema: schema},
	}
	params := tool.Parameters()
	require.NotEmpty(t, params)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(params, &parsed))
	assert.Equal(t, "object", parsed["type"])
}

func TestMCPProxyTool_ParametersEmpty(t *testing.T) {
	tool := &MCPProxyTool{
		toolDef: mcp.MCPToolDef{},
	}
	params := tool.Parameters()
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(params, &parsed))
	assert.Equal(t, "object", parsed["type"])
}

func TestRegisterMCPToolBridge_NoOp(t *testing.T) {
	registry := tools.NewRegistry()
	RegisterMCPToolBridge(registry)
	// No-op now, should not panic or register anything.
	all := registry.List()
	assert.Empty(t, all)
}
