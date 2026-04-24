# Tool Calling

Hera's agent loop includes a tool-calling engine that lets the LLM request actions — reading files, running commands, searching the web — and receive structured results back before generating a final response.

## How it works

1. You send a message.
2. The agent sends your message to the LLM with a list of available tool schemas.
3. The LLM decides whether a tool is needed. If so, it returns a tool call (name + arguments).
4. Hera executes the tool and appends the result to the conversation.
5. The LLM sees the result and either calls another tool or produces a final text response.
6. This loop repeats up to `agent.max_tool_calls` times.

```
User: "What is the latest Go release?"

LLM → tool_call: web_search(query="latest Go release")
Tool result: "Go 1.23.4, released December 2024..."
LLM → response: "The latest Go release is 1.23.4..."
```

## Tool approval

By default, potentially dangerous tools (shell execution, file writes, etc.) require approval before running. You'll see a prompt:

```
[APPROVAL REQUIRED]
Tool: shell
Args: {"command": "rm -rf /tmp/old-build"}

Allow? [y/N]
```

To auto-approve all tools (useful for automation):

```yaml
security:
  dangerous_approve: true
```

## Tool limits

The `agent.max_tool_calls` config key limits how many tool calls can happen in a single agent turn. The default is 20. This prevents runaway loops.

```yaml
agent:
  max_tool_calls: 50  # increase for complex tasks
```

## Available tools

All 40+ built-in tools are listed in the [Tool Reference](../../reference/tool-reference.md). Key tools include:

| Tool | Description |
|------|-------------|
| `read_file` | Read a file from disk |
| `write_file` | Write or overwrite a file |
| `patch` | Apply line-range edits to a file |
| `shell` | Run a shell command |
| `web_search` | Search the web (Exa API) |
| `web_extract` | Extract text content from a URL |
| `memory` | Store, retrieve, or search memories |
| `execute_code` | Run code in a sandboxed environment |
| `vision` | Analyze an image |
| `git_tool` | Git operations |
| `database_tool` | Query databases |
| `delegate` | Spawn a sub-agent for a subtask |

## Custom tools

You can define your own tools in `config.yaml` without writing Go code. See [Custom Tools](./custom-tools.md).

## MCP tools

Connect any MCP-compatible server and all its tools become available to the agent. See [MCP Servers](./mcp-servers.md).

## Tool result format

Tool results are returned to the LLM as structured JSON. Each tool implements three methods:

- `Name()` — unique tool identifier used in LLM API calls
- `Description()` — shown to the LLM to decide when to use the tool
- `Parameters()` — JSON Schema describing accepted arguments

This matches the tool-use format for both OpenAI and Anthropic APIs, so Hera works with any provider that supports tool/function calling.
