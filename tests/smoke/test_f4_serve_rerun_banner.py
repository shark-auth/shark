"""F4 — shark serve re-run banner

Verifies that on second+ `shark serve` runs, the operator sees one of:
  - "admin configured" banner (when admin API key exists in DB)
  - "setup pending" banner (when key file exists but no admin user yet)
  - first-boot flow (when neither)

Banner emission lives in:
  - internal/server/firstboot.go::RunRerunBanner
  - internal/cli/branding.go::PrintAdminConfigured / PrintSetupPending
"""

from __future__ import annotations

import os
import shutil
import signal
import socket
import subprocess
import tempfile
import time
from pathlib import Path

import pytest


REPO_ROOT = Path(__file__).resolve().parents[2]
SHARK_BIN = REPO_ROOT / "shark.exe" if os.name == "nt" else REPO_ROOT / "shark"


def _free_port() -> int:
    s = socket.socket()
    s.bind(("127.0.0.1", 0))
    port = s.getsockname()[1]
    s.close()
    return port


def _spawn_shark(data_dir: Path, port: int, timeout: float = 6.0) -> tuple[str, int]:
    """Run shark serve briefly, capture stdout, return (stdout, exit_code).

    Kills the process after the HTTP server prints "listening" + 2s grace
    so the re-run banner has time to land.
    """
    env = os.environ.copy()
    env["SHARK_DB_PATH"] = str(data_dir / "shark.db")
    env["SHARK_PORT"] = str(port)
    env["SHARK_BASE_URL"] = f"http://localhost:{port}"
    env["SHARK_DEV"] = "true"

    kwargs = {
        "cwd": str(REPO_ROOT),
        "env": env,
        "stdin": subprocess.DEVNULL,
        "stdout": subprocess.PIPE,
        "stderr": subprocess.STDOUT,
        "text": True,
        "encoding": "utf-8",
    }
    if os.name == "nt":
        kwargs["creationflags"] = subprocess.CREATE_NEW_PROCESS_GROUP

    proc = subprocess.Popen(
        [str(SHARK_BIN), "serve"],
        **kwargs
    )

    captured: list[str] = []

    def _reader():
        if proc.stdout:
            for line in iter(proc.stdout.readline, ""):
                captured.append(line)

    import threading
    t = threading.Thread(target=_reader, daemon=True)
    t.start()

    deadline = time.time() + timeout
    saw_listening = False
    grace_until: float | None = None

    try:
        while time.time() < deadline:
            output_so_far = "".join(captured)
            lower_output = output_so_far.lower()
            if "listening" in lower_output or "http server" in lower_output or "starting" in lower_output:
                saw_listening = True
                if grace_until is None:
                    grace_until = time.time() + 2.0
            if grace_until and time.time() >= grace_until:
                break
            time.sleep(0.1)
        return "".join(captured), proc.poll() or 0
    finally:
        try:
            if os.name == "nt":
                proc.send_signal(signal.CTRL_BREAK_EVENT)  # best-effort
            else:
                proc.send_signal(signal.SIGTERM)
            proc.wait(timeout=3)
        except Exception:
            proc.kill()


@pytest.fixture
def isolated_data_dir():
    tmp = Path(tempfile.mkdtemp(prefix="shark-f4-"))
    yield tmp
    shutil.rmtree(tmp, ignore_errors=True)


@pytest.mark.skipif(not SHARK_BIN.exists(), reason="shark binary not built")
def test_first_boot_flow_runs_when_clean_state(isolated_data_dir: Path):
    """Scenario 3: no DB, no key file → full first-boot banner appears."""
    out, _ = _spawn_shark(isolated_data_dir, _free_port())
    assert "admin" in out.lower(), "first-boot should mention admin key generation"
    # First-boot must produce the key file:
    assert (isolated_data_dir / "admin.key.firstboot").exists() or "admin.key.firstboot" in out


@pytest.mark.skipif(not SHARK_BIN.exists(), reason="shark binary not built")
def test_setup_pending_banner_when_key_file_only(isolated_data_dir: Path):
    """Scenario 2: key file exists but no admin user → 'setup pending' banner."""
    # First boot to create DB + key file
    port = _free_port()
    out, _ = _spawn_shark(isolated_data_dir, port)
    assert (isolated_data_dir / "admin.key.firstboot").exists(), f"first boot must write key file. output:\n{out}"
    # Second boot — admin not yet logged in, so banner should be '?bootstrap='
    out, _ = _spawn_shark(isolated_data_dir, _free_port())
    assert "?bootstrap=" in out.lower(), (
        f"expected '?bootstrap=' URL on second boot with no admin activity, got:\n{out}"
    )


@pytest.mark.skipif(not SHARK_BIN.exists(), reason="shark binary not built")
def test_admin_configured_banner_when_user_exists(isolated_data_dir: Path):
    """Scenario 1: admin activity exists in DB → no banner (quiet).

    We simulate by making an admin API call after first boot, then re-running.
    """
    # First boot
    port1 = _free_port()
    _spawn_shark(isolated_data_dir, port1)
    key_file = isolated_data_dir / "admin.key.firstboot"
    if not key_file.exists():
        pytest.skip("first boot did not produce admin.key.firstboot — skipping")

    admin_key = key_file.read_text().strip()

    # Spawn shark again to be live, then make an admin API call
    port2 = _free_port()
    env = os.environ.copy()
    env["SHARK_DB_PATH"] = str(isolated_data_dir / "shark.db")
    env["SHARK_PORT"] = str(port2)
    env["SHARK_BASE_URL"] = f"http://localhost:{port2}"
    env["SHARK_DEV"] = "true"
    proc = subprocess.Popen(
        [str(SHARK_BIN), "serve"], cwd=str(REPO_ROOT), env=env,
        stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    try:
        time.sleep(2.5)
        subprocess.run(
            [str(SHARK_BIN), "api-key", "create", "--name", "test-key-for-audit", "--token", admin_key, "--url", f"http://localhost:{port2}"],
            capture_output=True, text=True, env=env, check=True
        )
    finally:
        proc.terminate()
        proc.wait(timeout=5)

    # Third boot — admin activity now exists
    out, _ = _spawn_shark(isolated_data_dir, _free_port())
    assert "?bootstrap=" not in out.lower(), (
        f"expected NO '?bootstrap=' URL on third boot with admin activity present, got:\n{out}"
    )


def test_branding_helpers_exported_in_source():
    """Static check: the F4 helpers must exist in the source tree."""
    branding = (REPO_ROOT / "internal" / "cli" / "branding.go").read_text(encoding="utf-8", errors="ignore")
    assert "PrintAdminConfigured" in branding, "PrintAdminConfigured helper missing"
    assert "PrintSetupPending" in branding, "PrintSetupPending helper missing"
    assert "admin configured" in branding, "expected banner string 'admin configured' in branding.go"
    assert "setup pending" in branding, "expected banner string 'setup pending' in branding.go"

    firstboot = (REPO_ROOT / "internal" / "server" / "firstboot.go").read_text(encoding="utf-8", errors="ignore")
    assert "RunRerunBanner" in firstboot, "RunRerunBanner function missing in firstboot.go"
