"""Venv bootstrap and pip install wrapper.

We isolate every pip install into a Hera-owned virtualenv so user-requested
packages never pollute the system Python. The venv is created lazily on
the first install attempt — a minimal hera-python-mcp session that only
runs ``python_exec`` against stdlib never even touches pip.
"""

from __future__ import annotations

import datetime
import subprocess
import sys
import venv as stdlib_venv
from pathlib import Path

from . import paths


def ensure_venv() -> Path:
    """Create the venv if missing, return the path to its python binary."""
    paths.ensure_dirs()
    if not paths.VENV_DIR.exists():
        # system-site-packages=False — strict isolation.
        builder = stdlib_venv.EnvBuilder(with_pip=True, clear=False)
        builder.create(str(paths.VENV_DIR))
    return _venv_python(paths.VENV_DIR)


def _venv_python(venv_dir: Path) -> Path:
    if sys.platform == "win32":
        return venv_dir / "Scripts" / "python.exe"
    return venv_dir / "bin" / "python"


def pip_install(package: str, timeout: float = 180.0) -> tuple[bool, str]:
    """Install a package into the venv. Returns (success, combined output).

    Audit-logs every attempt (success OR failure) so an operator can
    review what the agent installed weeks later.
    """
    py = ensure_venv()
    # --disable-pip-version-check reduces output noise.
    # --no-input refuses interactive prompts (we're headless).
    proc = subprocess.run(
        [
            str(py),
            "-m",
            "pip",
            "install",
            "--disable-pip-version-check",
            "--no-input",
            package,
        ],
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    combined = (proc.stdout or "") + (proc.stderr or "")
    _audit(package, proc.returncode == 0, combined)
    return proc.returncode == 0, combined


def _audit(package: str, ok: bool, output: str) -> None:
    paths.ensure_dirs()
    status = "OK" if ok else "FAIL"
    timestamp = datetime.datetime.utcnow().isoformat(timespec="seconds") + "Z"
    with paths.PIP_AUDIT_LOG.open("a", encoding="utf-8") as f:
        f.write(f"{timestamp} {status} {package}\n")
        # Keep output excerpt so a failing install is debuggable from
        # the audit log alone without going back to logs.
        for line in output.splitlines()[-5:]:
            f.write(f"  {line}\n")
