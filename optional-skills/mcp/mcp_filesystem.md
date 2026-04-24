---
name: mcp_filesystem
description: "MCP filesystem server"
version: "1.0"
trigger: "mcp filesystem file access read write"
platforms: []
requires_tools: ["run_command"]
---

# MCP filesystem server

## Purpose
MCP filesystem server for extending AI agent capabilities through specialized MCP server implementations.

## Instructions
1. Install and configure the MCP server
2. Set up the transport layer (stdio, HTTP/SSE)
3. Register tools and resources with the agent
4. Test all tool invocations and error handling
5. Deploy and monitor in production

## Server Configuration
- Define exposed tools with JSON Schema inputs
- Configure resource endpoints for data access
- Set up authentication if needed
- Configure logging and error reporting
- Define rate limits and usage policies

## Tool Design
- Write clear, concise tool descriptions for LLM selection
- Define strict input schemas with validation
- Return structured outputs that agents can parse
- Handle errors with descriptive messages
- Support cancellation for long-running operations

## Deployment
- Package as Docker container for consistent environments
- Configure health checks and monitoring
- Set up log aggregation for debugging
- Implement graceful shutdown handling
- Use environment variables for configuration

## Best Practices
- Keep tools focused on single responsibilities
- Validate all inputs at the server boundary
- Use descriptive error messages that help the LLM
- Implement proper timeout handling
- Version the server API for backward compatibility
