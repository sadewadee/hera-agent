# MCP Servers

Hera can connect to external Model Context Protocol (MCP) servers and make all their tools available to the agent. This lets you extend Hera with any MCP-compatible tool server.

## Configuration

Add MCP servers to `~/.hera/config.yaml`:

```yaml
mcp_servers:
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/projects"]

  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: ${GITHUB_TOKEN}

  - name: postgres
    command: npx
    args: ["-y", "@modelcontextprotocol/server-postgres"]
    env:
      POSTGRES_URL: postgresql://localhost/mydb

  - name: custom-server
    command: /usr/local/bin/my-mcp-server
    args: ["--port", "8080"]
    env:
      MY_API_KEY: ${MY_API_KEY}
```

### MCPServerEntry fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Identifier for this server (used in logs) |
| `command` | string | Command to start the server |
| `args` | string[] | Arguments passed to the command |
| `env` | map | Environment variables for the server process |

## How it works

When Hera starts, it launches each configured MCP server as a subprocess communicating over stdio. The server advertises its tool list, and Hera registers each tool under the `mcp_tool` wrapper. The agent can then call any MCP tool like any other Hera tool.

## Using MCP tools

MCP tools appear in the agent's toolbox automatically. The agent can discover them:

```
You: What tools do you have from the filesystem MCP server?

Hera: I have access to these filesystem tools:
- read_file: Read file contents
- write_file: Write to a file
- list_directory: List directory contents
- search_files: Search for files by pattern
```

You can also call them directly:

```
You: List the contents of /home/user/projects using the filesystem server

Hera: [using mcp_tool: filesystem/list_directory]
projects/
  hera/
  my-app/
  scripts/
```

## Common MCP servers

| Server | npm Package | Description |
|--------|-------------|-------------|
| Filesystem | `@modelcontextprotocol/server-filesystem` | Read/write local files |
| GitHub | `@modelcontextprotocol/server-github` | GitHub API operations |
| PostgreSQL | `@modelcontextprotocol/server-postgres` | Query PostgreSQL |
| SQLite | `@modelcontextprotocol/server-sqlite` | Query SQLite databases |
| Brave Search | `@modelcontextprotocol/server-brave-search` | Web search |
| Puppeteer | `@modelcontextprotocol/server-puppeteer` | Browser automation |
| Memory | `@modelcontextprotocol/server-memory` | Knowledge graph memory |

## Hera as an MCP server

Hera itself can act as an MCP server, exposing its tools to IDEs like Cursor and VS Code. See [MCP Protocol](../../developer-guide/mcp-protocol.md) for setup instructions.

## Troubleshooting

**Server won't start:** Check that the command and args are correct. Hera logs MCP server startup errors at startup.

**Tools not appearing:** The MCP server must successfully advertise tools via the `tools/list` method. Verify your server is functioning correctly.

**Authentication errors:** Ensure environment variables like API keys are correctly set in the `env` map.
