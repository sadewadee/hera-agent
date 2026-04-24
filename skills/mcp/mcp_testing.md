---
name: mcp_testing
description: "MCP server testing and validation"
version: "1.0"
trigger: "mcp testing validation server test"
platforms: []
requires_tools: ["run_command"]
---

# MCP Server Testing

## Purpose
Test MCP server implementations for correctness, reliability, and performance.

## Instructions
1. Set up test harness for MCP protocol
2. Test tool schema validation and execution
3. Test resource endpoints
4. Test error handling and edge cases
5. Performance test under load

## Test Areas
- Protocol compliance (JSON-RPC format)
- Tool input validation
- Tool execution correctness
- Resource access and pagination
- Error handling and recovery
- Concurrent request handling

## Best Practices
- Test both valid and invalid inputs
- Verify error messages are descriptive
- Test timeout and cancellation behavior
- Benchmark latency under load
- Test with the actual LLM client
