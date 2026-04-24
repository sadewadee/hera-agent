package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sadewadee/hera/internal/mcp"
	"github.com/sadewadee/hera/internal/tools"
)

// MCPProxyTool wraps a tool from an external MCP server so it can be
// used as a Hera tool. The backing client is a *ManagedClient which
// handles the on_demand lifecycle — subprocess killed after idle,
// respawned on next call — without proxy tools needing to care.
type MCPProxyTool struct {
	client     *mcp.ManagedClient
	toolDef    mcp.MCPToolDef
	serverName string
}

func (t *MCPProxyTool) Name() string {
	// Prefix with server name to avoid conflicts: "mcp__servername__toolname"
	return fmt.Sprintf("mcp__%s__%s", t.serverName, t.toolDef.Name)
}

func (t *MCPProxyTool) Description() string {
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, t.toolDef.Description)
}

func (t *MCPProxyTool) Parameters() json.RawMessage {
	if len(t.toolDef.InputSchema) > 0 {
		return t.toolDef.InputSchema
	}
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t *MCPProxyTool) Execute(ctx context.Context, args json.RawMessage) (*tools.Result, error) {
	result, err := t.client.CallTool(ctx, t.toolDef.Name, args)
	if err != nil {
		return &tools.Result{
			Content: fmt.Sprintf("MCP tool %s error: %v", t.toolDef.Name, err),
			IsError: true,
		}, nil
	}

	if len(result) > 50000 {
		result = result[:50000] + "\n... (truncated)"
	}

	return &tools.Result{Content: result}, nil
}

// RegisterMCPTools connects to MCP servers and registers their tools
// with the Hera registry. Each server is wrapped in a ManagedClient so
// on_demand mode (default) kills the subprocess after idle and
// respawns on next call. Returns the managed clients so they can be
// cleanly closed at shutdown.
func RegisterMCPTools(registry *tools.Registry, servers []mcp.MCPServerConfig) []*mcp.ManagedClient {
	var clients []*mcp.ManagedClient

	for _, cfg := range servers {
		client, err := mcp.NewManagedClient(cfg)
		if err != nil {
			fmt.Printf("  [MCP] failed to connect to %s: %v\n", cfg.Name, err)
			continue
		}

		clients = append(clients, client)

		for _, tool := range client.Tools() {
			registry.Register(&MCPProxyTool{
				client:     client,
				toolDef:    tool,
				serverName: cfg.Name,
			})
		}

		fmt.Printf("  [MCP] %s: %d tools registered (mode=%s)\n",
			cfg.Name, len(client.Tools()), client.Mode())
	}

	return clients
}

// RegisterMCPToolBridge kept for backward compatibility (no-op if MCP clients are used).
func RegisterMCPToolBridge(registry *tools.Registry) {
	// No-op: MCP tools are now registered via RegisterMCPTools.
}
