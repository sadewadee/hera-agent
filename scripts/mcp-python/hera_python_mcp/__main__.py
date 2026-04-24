"""Entry point: ``python -m hera_python_mcp`` or ``hera-python-mcp``.

Launches the MCP stdio server. The Hera Go binary spawns this process
when ``mcp_servers.python.enabled`` is true in config.yaml.
"""

from __future__ import annotations

import asyncio
import logging
import sys

from . import server


def main() -> int:
    # Send logs to stderr; stdout is reserved for MCP JSON-RPC framing.
    logging.basicConfig(
        level=logging.INFO,
        stream=sys.stderr,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    try:
        asyncio.run(server.run())
    except KeyboardInterrupt:
        return 130
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
