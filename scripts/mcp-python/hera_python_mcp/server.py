"""MCP server wiring — exposes the Python tools to Hera via stdio.

Uses the official ``mcp`` SDK from Anthropic. Tool handlers dispatch
into exec_tool/registry/venv modules and return JSON-serialisable
results that the Go side receives as plain tool responses.
"""

from __future__ import annotations

import json
import logging
from typing import Any

from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.types import TextContent, Tool

from . import exec_tool, paths, registry, venv as venv_mod


logger = logging.getLogger("hera_python_mcp")


def _native_tools() -> list[Tool]:
    """Built-in tool definitions. Registered tools are appended later."""
    return [
        Tool(
            name="python_exec",
            description=(
                "Run a one-shot Python snippet in the Hera-managed venv. "
                "Returns stdout, stderr, exit code, and a timed_out flag. "
                "Useful for ad-hoc calculation, data processing, or "
                "quick integrations. For reusable logic, prefer "
                "python_register_tool instead."
            ),
            inputSchema={
                "type": "object",
                "properties": {
                    "code": {
                        "type": "string",
                        "description": "Python source to execute.",
                    },
                    "timeout": {
                        "type": "number",
                        "description": "Seconds before the subprocess is killed (default 30, max 300).",
                    },
                },
                "required": ["code"],
            },
        ),
        Tool(
            name="python_register_tool",
            description=(
                "Register a persistent Python tool the agent can call "
                "by name on subsequent turns. The code must define a "
                "top-level `run(**kwargs)` function that returns a "
                "JSON-serialisable value. Tool appears as `py_<name>` "
                "after registration. Call python_list_tools to verify."
            ),
            inputSchema={
                "type": "object",
                "properties": {
                    "name": {
                        "type": "string",
                        "description": "File-safe slug: lowercase alphanumeric + dash/underscore, <=64 chars.",
                    },
                    "description": {
                        "type": "string",
                        "description": "One-line summary shown to the LLM when choosing tools.",
                    },
                    "params_schema": {
                        "type": "object",
                        "description": "JSON Schema describing the `run` kwargs.",
                    },
                    "code": {
                        "type": "string",
                        "description": "Python source defining `def run(**kwargs):`.",
                    },
                },
                "required": ["name", "description", "code"],
            },
        ),
        Tool(
            name="python_list_tools",
            description="List all registered Python tools with their descriptions.",
            inputSchema={"type": "object", "properties": {}},
        ),
        Tool(
            name="python_unregister_tool",
            description="Delete a registered Python tool by name.",
            inputSchema={
                "type": "object",
                "properties": {
                    "name": {"type": "string"},
                },
                "required": ["name"],
            },
        ),
        Tool(
            name="python_pip_install",
            description=(
                "Install a package into the Hera-managed venv. Audit "
                "logged to ~/.hera/python/pip-audit.log. Use when a "
                "registered tool or python_exec snippet needs a "
                "third-party dependency."
            ),
            inputSchema={
                "type": "object",
                "properties": {
                    "package": {
                        "type": "string",
                        "description": "pip-style requirement, e.g. 'requests' or 'pandas==2.2.0'.",
                    },
                },
                "required": ["package"],
            },
        ),
    ]


def _tool_for_registered(rt: registry.RegisteredTool) -> Tool:
    return Tool(
        name=rt.mcp_name,
        description=rt.description,
        inputSchema=rt.params_schema,
    )


def _as_text(payload: Any) -> list[TextContent]:
    """Serialise any tool response as a JSON TextContent — the MCP
    pattern that works across all clients including Hera's Go one."""
    if isinstance(payload, str):
        return [TextContent(type="text", text=payload)]
    return [TextContent(type="text", text=json.dumps(payload, default=str, indent=2))]


async def run() -> None:
    paths.ensure_dirs()
    server: Server = Server("hera-python-mcp")

    @server.list_tools()
    async def list_tools() -> list[Tool]:
        tools = _native_tools()
        for rt in registry.load_all():
            tools.append(_tool_for_registered(rt))
        return tools

    @server.call_tool()
    async def call_tool(name: str, arguments: dict[str, Any]) -> list[TextContent]:
        # Registered tools — dispatch through the registry.
        if name.startswith(registry.MCP_PREFIX):
            short = name[len(registry.MCP_PREFIX) :]
            rts = [rt for rt in registry.load_all() if rt.name == short]
            if not rts:
                return _as_text({"error": f"tool {name!r} not found"})
            result = registry.invoke(rts[0], arguments or {})
            return _as_text(result)

        # Native tools.
        if name == "python_exec":
            code = (arguments or {}).get("code", "")
            timeout = (arguments or {}).get("timeout")
            return _as_text(exec_tool.run(code, timeout))

        if name == "python_register_tool":
            a = arguments or {}
            try:
                rt = registry.register(
                    name=a.get("name", ""),
                    description=a.get("description", ""),
                    params_schema=a.get("params_schema"),
                    code=a.get("code", ""),
                )
            except ValueError as exc:
                return _as_text({"error": str(exc)})
            return _as_text(
                {
                    "registered": rt.mcp_name,
                    "description": rt.description,
                    "hint": "call python_list_tools to confirm; the tool is now available to the agent.",
                }
            )

        if name == "python_list_tools":
            return _as_text(
                {
                    "tools": [
                        {"mcp_name": rt.mcp_name, "description": rt.description}
                        for rt in registry.load_all()
                    ]
                }
            )

        if name == "python_unregister_tool":
            removed = registry.unregister((arguments or {}).get("name", ""))
            return _as_text({"removed": removed})

        if name == "python_pip_install":
            pkg = (arguments or {}).get("package", "")
            if not pkg:
                return _as_text({"error": "package is required"})
            ok, out = venv_mod.pip_install(pkg)
            return _as_text({"installed": ok, "output_tail": out[-2000:]})

        return _as_text({"error": f"unknown tool {name!r}"})

    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream, write_stream, server.create_initialization_options()
        )
