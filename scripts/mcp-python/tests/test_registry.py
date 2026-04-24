"""Unit tests for the Python tool registry.

Covers slug validation, round-trip persistence, invocation of a
registered tool, and unregister cleanup. Each test points
HERA_PYTHON_HOME at a tmpdir so nothing pollutes ~/.hera/python.
"""

from __future__ import annotations

import importlib
import json

import pytest


@pytest.fixture(autouse=True)
def _isolated_home(tmp_path, monkeypatch):
    monkeypatch.setenv("HERA_PYTHON_HOME", str(tmp_path))
    # Force module reload so paths pick up the new env var.
    import hera_python_mcp.paths as paths_mod

    importlib.reload(paths_mod)
    import hera_python_mcp.registry as registry_mod

    importlib.reload(registry_mod)
    yield


def test_sanitize_name_accepts_valid_slugs():
    from hera_python_mcp import registry

    assert registry.sanitize_name("my_tool") == "my_tool"
    assert registry.sanitize_name("tool-1") == "tool-1"
    assert registry.sanitize_name("  Trimmed  ") == "trimmed"


def test_sanitize_name_rejects_invalid():
    from hera_python_mcp import registry

    assert registry.sanitize_name("has spaces") is None
    assert registry.sanitize_name("Upper!Case") is None
    assert registry.sanitize_name("") is None
    # Over 64 chars.
    assert registry.sanitize_name("x" * 65) is None


def test_register_persists_and_load_all_reads_back():
    from hera_python_mcp import registry

    rt = registry.register(
        name="echo",
        description="echoes the input message",
        params_schema={"type": "object", "properties": {"msg": {"type": "string"}}},
        code="def run(**kwargs):\n    return kwargs\n",
    )
    assert rt.mcp_name == "py_echo"

    all_tools = registry.load_all()
    assert len(all_tools) == 1
    assert all_tools[0].name == "echo"
    assert all_tools[0].description == "echoes the input message"
    assert all_tools[0].params_schema["properties"]["msg"]["type"] == "string"


def test_register_rejects_invalid_name():
    from hera_python_mcp import registry

    with pytest.raises(ValueError, match="name must match"):
        registry.register(
            name="Has Spaces",
            description="x",
            params_schema={},
            code="def run(**k): return k",
        )


def test_register_requires_run_function():
    from hera_python_mcp import registry

    with pytest.raises(ValueError, match="top-level"):
        registry.register(
            name="broken",
            description="missing run",
            params_schema={},
            code="def not_run(**k): return k",
        )


def test_unregister_removes_files():
    from hera_python_mcp import registry, paths

    registry.register(
        name="doomed",
        description="will be removed",
        params_schema={},
        code="def run(**k): return k",
    )
    assert (paths.TOOLS_DIR / "doomed.py").exists()

    assert registry.unregister("doomed") is True
    assert not (paths.TOOLS_DIR / "doomed.py").exists()
    assert not (paths.TOOLS_DIR / "doomed.json").exists()

    # Second unregister is a no-op, not an error.
    assert registry.unregister("doomed") is False


def test_invoke_runs_the_tool_in_venv():
    from hera_python_mcp import registry

    rt = registry.register(
        name="adder",
        description="adds two numbers",
        params_schema={
            "type": "object",
            "properties": {"a": {"type": "number"}, "b": {"type": "number"}},
            "required": ["a", "b"],
        },
        code="def run(a, b, **_):\n    return {'sum': a + b}\n",
    )
    result = registry.invoke(rt, {"a": 3, "b": 4}, timeout=30.0)
    assert "result" in result
    assert result["result"] == {"sum": 7}


def test_invoke_surfaces_tool_exception():
    from hera_python_mcp import registry

    rt = registry.register(
        name="boom",
        description="always errors",
        params_schema={"type": "object"},
        code="def run(**_):\n    raise RuntimeError('nope')\n",
    )
    result = registry.invoke(rt, {}, timeout=10.0)
    assert result.get("error") == "non-zero exit"
    assert "RuntimeError" in (result.get("stderr") or "")


def test_mcp_prefix_applied_consistently():
    from hera_python_mcp import registry

    rt = registry.register(
        name="prefixed",
        description="x",
        params_schema={},
        code="def run(**k): return 1",
    )
    assert rt.mcp_name.startswith("py_")
    assert rt.mcp_name == "py_prefixed"
