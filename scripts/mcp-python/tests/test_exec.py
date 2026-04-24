"""Unit tests for python_exec.

Exercises stdout/stderr capture, exit codes, timeout handling, and
empty-input guards. Uses the venv bootstrap so we cover that too.
"""

from __future__ import annotations

import importlib

import pytest


@pytest.fixture(autouse=True)
def _isolated_home(tmp_path, monkeypatch):
    monkeypatch.setenv("HERA_PYTHON_HOME", str(tmp_path))
    import hera_python_mcp.paths as paths_mod

    importlib.reload(paths_mod)
    import hera_python_mcp.venv as venv_mod

    importlib.reload(venv_mod)
    import hera_python_mcp.exec_tool as exec_mod

    importlib.reload(exec_mod)
    yield


def test_exec_captures_stdout():
    from hera_python_mcp import exec_tool

    out = exec_tool.run("print('hello from hera')")
    assert out["exit_code"] == 0
    assert "hello from hera" in out["stdout"]
    assert out["timed_out"] is False


def test_exec_captures_stderr_and_nonzero_exit():
    from hera_python_mcp import exec_tool

    out = exec_tool.run("import sys; sys.stderr.write('boom'); sys.exit(7)")
    assert out["exit_code"] == 7
    assert "boom" in out["stderr"]


def test_exec_empty_code_returns_error():
    from hera_python_mcp import exec_tool

    out = exec_tool.run("")
    assert out["exit_code"] == 2
    assert "empty" in out["stderr"].lower()


def test_exec_honours_timeout():
    from hera_python_mcp import exec_tool

    out = exec_tool.run("import time; time.sleep(5)", timeout=0.5)
    assert out["timed_out"] is True
    assert out["exit_code"] == -1


def test_exec_default_timeout_used_on_zero():
    # Regression: timeout=0 should be treated as "use default", not
    # "kill immediately".
    from hera_python_mcp import exec_tool

    out = exec_tool.run("print('ok')", timeout=0)
    assert out["timed_out"] is False
    assert out["exit_code"] == 0
