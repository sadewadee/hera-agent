---
name: mcp_client
description: "MCP client integration"
version: "1.0"
trigger: "mcp client connect consume tools"
platforms: []
requires_tools: ["run_command"]
---

# MCP client integration

## Purpose
MCP client integration for extending AI agent capabilities through the Model Context Protocol.

## Instructions
1. Understand the MCP architecture and protocol
2. Design the server/client interaction pattern
3. Implement the appropriate MCP components
4. Test communication and error handling
5. Deploy and monitor the MCP integration

## MCP Architecture
- **Servers**: Expose tools, resources, and prompts to AI agents
- **Clients**: Connect to servers and invoke capabilities
- **Transports**: Communication channels (stdio, HTTP/SSE)
- **Tools**: Functions that agents can call
- **Resources**: Data sources agents can read
- **Prompts**: Template prompts for common interactions

## Implementation
- Use the MCP SDK for your language (TypeScript, Python)
- Define tool schemas with JSON Schema for input validation
- Handle errors gracefully with descriptive messages
- Support cancellation for long-running operations
- Log all tool invocations for debugging

## Protocol Details
- JSON-RPC 2.0 message format
- Capability negotiation during initialization
- Streaming support for long responses
- Progress reporting for multi-step operations

## Best Practices
- Define clear tool descriptions for LLM selection
- Validate all inputs at the server boundary
- Implement proper timeout handling
- Use descriptive error messages that help the LLM retry
- Version your MCP server API for backward compatibility
