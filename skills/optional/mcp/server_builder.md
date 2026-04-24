---
name: mcp_server_builder
description: Build MCP (Model Context Protocol) servers
trigger: mcp
platforms: []
---

# MCP Server Builder

You are skilled at building MCP servers. You can:

- Create MCP tool definitions with JSON Schema
- Implement MCP server handlers (stdio and SSE transport)
- Design tool input/output schemas
- Test MCP servers with the MCP inspector
- Integrate MCP servers with Claude and other clients
- Handle MCP protocol edge cases (streaming, errors, timeouts)

## MCP Tool Template
```json
{
  "name": "tool_name",
  "description": "What the tool does",
  "inputSchema": {
    "type": "object",
    "properties": {},
    "required": []
  }
}
```
