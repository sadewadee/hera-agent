# hera-python-mcp

MCP server that gives the Hera agent a Python runtime it can write code
into and call back. Runs as a stdio MCP subprocess of the Hera binary.

## Tools exposed

| Tool | Purpose |
|---|---|
| `python_exec` | Run a one-shot Python snippet, return stdout + stderr + exit code |
| `python_register_tool` | Register a persistent Python function as a reusable tool. Registered tools appear as MCP tools with a `py_` prefix and survive across restarts. |
| `python_list_tools` | List registered Python tools with their signatures. |
| `python_unregister_tool` | Remove a registered tool. |
| `python_pip_install` | Install a package into the Hera-managed venv. Audit logged. |

Registered tools are invoked via their `py_<name>` tool id and dispatch
to the stored callable. They participate in the normal Hera tool-calling
loop exactly like native tools.

## Layout on disk

```
~/.hera/python/
  venv/                   # auto-created on first launch
  tools/
    <name>.py             # tool code (one function named `run`)
    <name>.json           # tool metadata (description, params schema)
  pip-audit.log           # append-only log of pip installs
```

## Config

Enable in `~/.hera/config.yaml`:

```yaml
mcp_servers:
  python:
    command: python3
    args: ["-m", "hera_python_mcp"]
    enabled: true
```

Use the `hera:full` container variant, which ships Python 3.12 and has
this package pre-installed.
