---
name: mcp_docker
description: "MCP Docker container server"
version: "1.0"
trigger: "mcp docker container management"
platforms: []
requires_tools: ["run_command"]
---

# MCP Docker Server

## Purpose
Manage Docker containers through MCP protocol for AI agent container orchestration.

## Instructions
1. Set up MCP Docker server
2. Define container management tools
3. Configure access controls
4. Implement container lifecycle operations
5. Monitor container health

## Operations
- List running containers
- Start/stop containers
- Pull and build images
- Inspect container logs
- Manage Docker networks and volumes

## Best Practices
- Use minimal base images for security
- Implement proper resource limits
- Log container output for debugging
- Use health checks for reliability
- Clean up unused containers and images
