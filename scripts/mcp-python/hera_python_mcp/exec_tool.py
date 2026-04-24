"""python_exec: run a one-shot Python snippet.

Code runs in a subprocess using the Hera-managed venv's interpreter, so
imports of anything installed via python_pip_install are available but
system Python is NOT. Captures stdout/stderr separately; honours a
timeout. Result shape is JSON-safe so MCP can serialize it directly.
"""

from __future__ import annotations

import subprocess
from typing import Any

from . import venv as venv_mod


DEFAULT_TIMEOUT_S = 30.0
MAX_TIMEOUT_S = 300.0


def run(code: str, timeout: float | None = None) -> dict[str, Any]:
    """Execute ``code`` as Python and return result payload.

    Args:
        code: full Python source, evaluated as a fresh module.
        timeout: seconds before the subprocess is killed. Capped at
            MAX_TIMEOUT_S so runaway loops don't block the agent turn.

    Returns:
        dict with stdout, stderr, exit_code, timed_out fields.
    """
    if not code or not code.strip():
        return {
            "stdout": "",
            "stderr": "empty code",
            "exit_code": 2,
            "timed_out": False,
        }

    tmo = timeout if timeout is not None else DEFAULT_TIMEOUT_S
    if tmo <= 0 or tmo > MAX_TIMEOUT_S:
        tmo = DEFAULT_TIMEOUT_S

    py = venv_mod.ensure_venv()

    try:
        proc = subprocess.run(
            [str(py), "-c", code],
            capture_output=True,
            text=True,
            timeout=tmo,
        )
    except subprocess.TimeoutExpired as te:
        return {
            "stdout": te.stdout.decode(errors="replace") if te.stdout else "",
            "stderr": te.stderr.decode(errors="replace") if te.stderr else "",
            "exit_code": -1,
            "timed_out": True,
        }

    return {
        "stdout": proc.stdout or "",
        "stderr": proc.stderr or "",
        "exit_code": proc.returncode,
        "timed_out": False,
    }
