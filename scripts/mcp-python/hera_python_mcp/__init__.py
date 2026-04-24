"""Hera Python MCP server.

Exposes a Python execution sandbox and a registry of persistent Python
tools to the Hera Go agent via the MCP protocol. The Go side treats
this as just another MCP server — no special wiring.
"""

__version__ = "0.10.0"
