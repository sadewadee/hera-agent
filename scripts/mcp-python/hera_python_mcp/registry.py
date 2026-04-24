"""Persistent registry of Python-defined tools.

A registered tool is a Python source file plus a JSON metadata sidecar:

    ~/.hera/python/tools/<name>.py    — contains a ``run(**kwargs)`` callable
    ~/.hera/python/tools/<name>.json  — {"description", "params_schema"}

At startup the registry scans ``tools/`` and exposes each one to the
MCP server as a tool named ``py_<name>`` so the agent can distinguish
Python-registered tools from built-ins at a glance.

Invocation runs the file as a subprocess in the venv interpreter,
passing arguments as JSON over stdin. This matches how ``python_exec``
works, keeps the host process clean, and means a misbehaving tool can
only crash itself.
"""

from __future__ import annotations

import json
import re
import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from . import paths, venv as venv_mod


# Tool names are file-path safe so we can map directly onto disk.
NAME_RE = re.compile(r"^[a-z0-9_-]{1,64}$")

# Prefix applied to registered tools at the MCP layer so the agent can
# tell them apart from built-in tools.
MCP_PREFIX = "py_"


@dataclass
class RegisteredTool:
    name: str
    description: str
    params_schema: dict[str, Any]
    code_path: Path

    @property
    def mcp_name(self) -> str:
        return MCP_PREFIX + self.name


def sanitize_name(name: str) -> str | None:
    """Return name if valid, else None. Caller error-surfaces."""
    n = (name or "").strip().lower()
    if not NAME_RE.match(n):
        return None
    return n


def register(
    name: str,
    description: str,
    params_schema: dict[str, Any] | None,
    code: str,
) -> RegisteredTool:
    """Persist a tool to disk. Overwrites if it already exists."""
    n = sanitize_name(name)
    if n is None:
        raise ValueError("name must match [a-z0-9_-]{1,64}; got %r" % name)
    if not code or not code.strip():
        raise ValueError("code is required")
    if "def run(" not in code:
        raise ValueError("code must define a top-level `run(**kwargs)` function")

    paths.ensure_dirs()
    code_path = paths.TOOLS_DIR / f"{n}.py"
    meta_path = paths.TOOLS_DIR / f"{n}.json"
    code_path.write_text(code, encoding="utf-8")
    meta_path.write_text(
        json.dumps(
            {
                "description": description or "",
                "params_schema": params_schema or {"type": "object"},
            },
            indent=2,
        ),
        encoding="utf-8",
    )
    return RegisteredTool(
        name=n,
        description=description or "",
        params_schema=params_schema or {"type": "object"},
        code_path=code_path,
    )


def unregister(name: str) -> bool:
    """Delete a registered tool. Returns True if something was removed."""
    n = sanitize_name(name)
    if n is None:
        return False
    code_path = paths.TOOLS_DIR / f"{n}.py"
    meta_path = paths.TOOLS_DIR / f"{n}.json"
    removed = False
    for p in (code_path, meta_path):
        if p.exists():
            p.unlink()
            removed = True
    return removed


def load_all() -> list[RegisteredTool]:
    """Scan ``tools/`` for (name.py, name.json) pairs and return them."""
    paths.ensure_dirs()
    out: list[RegisteredTool] = []
    for py in sorted(paths.TOOLS_DIR.glob("*.py")):
        name = py.stem
        if not NAME_RE.match(name):
            continue
        meta_path = py.with_suffix(".json")
        description = ""
        schema: dict[str, Any] = {"type": "object"}
        if meta_path.exists():
            try:
                meta = json.loads(meta_path.read_text(encoding="utf-8"))
                description = str(meta.get("description", ""))
                s = meta.get("params_schema")
                if isinstance(s, dict):
                    schema = s
            except (json.JSONDecodeError, OSError):
                # Malformed sidecar — expose the tool with empty metadata
                # rather than hide it. Operator can fix or delete.
                pass
        out.append(
            RegisteredTool(
                name=name,
                description=description,
                params_schema=schema,
                code_path=py,
            )
        )
    return out


def invoke(
    tool: RegisteredTool, kwargs: dict[str, Any], timeout: float = 60.0
) -> dict[str, Any]:
    """Run a registered tool's ``run(**kwargs)`` and return its result.

    Arguments are piped as JSON on stdin. The tool code reads them with
    ``json.load(sys.stdin)`` (see the bootstrap wrapper below). Return
    value is whatever JSON the tool writes to stdout on its last line.
    """
    py = venv_mod.ensure_venv()

    # Wrapper reads args, calls `run`, prints the JSON-encoded result.
    wrapper = f"""
import json, sys, runpy
ns = runpy.run_path({str(tool.code_path)!r})
args = json.load(sys.stdin)
result = ns['run'](**args)
print(json.dumps(result, default=str))
""".strip()

    try:
        proc = subprocess.run(
            [str(py), "-c", wrapper],
            input=json.dumps(kwargs or {}),
            capture_output=True,
            text=True,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired as te:
        return {
            "error": "timeout",
            "stdout": te.stdout.decode(errors="replace") if te.stdout else "",
            "stderr": te.stderr.decode(errors="replace") if te.stderr else "",
        }

    if proc.returncode != 0:
        return {
            "error": "non-zero exit",
            "exit_code": proc.returncode,
            "stdout": proc.stdout,
            "stderr": proc.stderr,
        }

    # Tool output goes to stdout; the last line must be JSON. Anything
    # before is diagnostic print(). Return both so the agent can surface
    # the useful result and the operator can see auxiliary output.
    lines = (proc.stdout or "").splitlines()
    payload: Any = None
    if lines:
        try:
            payload = json.loads(lines[-1])
        except json.JSONDecodeError:
            payload = {"raw": proc.stdout}

    return {
        "result": payload,
        "stderr": proc.stderr or "",
    }
