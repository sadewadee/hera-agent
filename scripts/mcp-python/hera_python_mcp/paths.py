"""Resolve the on-disk paths this server owns.

Everything lives under ``~/.hera/python/``. The directory is created on
first launch — users do not need to pre-seed anything. Paths are
resolved once at import so tests can override via the HERA_PYTHON_HOME
environment variable without races.
"""

from __future__ import annotations

import os
from pathlib import Path


def _resolve_root() -> Path:
    """Return ``~/.hera/python/`` or the HERA_PYTHON_HOME override.

    Tests set ``HERA_PYTHON_HOME`` to a tmpdir so they don't touch the
    real dotfile. Production just uses the default.
    """
    override = os.environ.get("HERA_PYTHON_HOME")
    if override:
        return Path(override)
    return Path.home() / ".hera" / "python"


ROOT: Path = _resolve_root()
VENV_DIR: Path = ROOT / "venv"
TOOLS_DIR: Path = ROOT / "tools"
PIP_AUDIT_LOG: Path = ROOT / "pip-audit.log"


def ensure_dirs() -> None:
    """Idempotent mkdir for all the paths this server writes to."""
    ROOT.mkdir(parents=True, exist_ok=True)
    TOOLS_DIR.mkdir(parents=True, exist_ok=True)
