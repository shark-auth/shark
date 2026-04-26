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
    env["SHARK_DATA_DIR"] = str(data_dir)
    env["SHARK_PORT"] = str(port)
    env["SHARK_BASE_URL"] = f"http://localhost:{port}"
    env["SHARK_DEV"] = "true"

    proc = subprocess.Popen(
        [str(SHARK_BIN), "serve"],
        cwd=str(REPO_ROOT),
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )

    captured: list[str] = []
    deadline = time.time() + timeout
    saw_listening = False
    grace_until: float | None = None

    try:
        while time.time() < deadline:
            line = proc.stdout.readline() if proc.stdout else ""
            if not line:
                if proc.poll() is not None:
                    break
                time.sleep(0.05)
                continue
            captured.append(line)
            if "listening" in line.lower() or "http server" in line.lower():
                saw_listening = True
                grace_until = time.time() + 2.0
            if grace_until and time.time() >= grace_until:
                break
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
    _spawn_shark(isolated_data_dir, port)
    assert (isolated_data_dir / "admin.key.firstboot").exists(), "first boot must write key file"
    # Second boot — admin not yet created via /admin/setup, so banner should be 'setup pending'
    out, _ = _spawn_shark(isolated_data_dir, _free_port())
    assert "setup pending" in out.lower() or "key in:" in out.lower(), (
        f"expected 'setup pending' banner on second boot with no admin user, got:\n{out}"
    )


@pytest.mark.skipif(not SHARK_BIN.exists(), reason="shark binary not built")
def test_admin_configured_banner_when_user_exists(isolated_data_dir: Path):
    """Scenario 1: admin user exists in DB → 'admin configured' banner.

    We simulate by creating an admin via /admin/setup after first boot, then re-running.
    """
    # First boot
    port1 = _free_port()
    _spawn_shark(isolated_data_dir, port1)
    key_file = isolated_data_dir / "admin.key.firstboot"
    if not key_file.exists():
        pytest.skip("first boot did not produce admin.key.firstboot — skipping")

    # Spawn shark again to be live, then call /admin/setup to mint admin user
    port2 = _free_port()
    env = os.environ.copy()
    env["SHARK_DATA_DIR"] = str(isolated_data_dir)
    env["SHARK_PORT"] = str(port2)
    env["SHARK_BASE_URL"] = f"http://localhost:{port2}"
    env["SHARK_DEV"] = "true"
    proc = subprocess.Popen(
        [str(SHARK_BIN), "serve"], cwd=str(REPO_ROOT), env=env,
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    try:
        time.sleep(2.5)
        import urllib.request, urllib.error, json
        admin_key = key_file.read_text().strip()
        try:
            req = urllib.request.Request(
                f"http://localhost:{port2}/admin/setup",
                data=json.dumps({"email": "admin@example.com", "password": "F4TestPass!23"}).encode(),
                headers={"Authorization": f"Bearer {admin_key}", "Content-Type": "application/json"},
                method="POST",
            )
            urllib.request.urlopen(req, timeout=4)
        except urllib.error.URLError:
            pytest.skip("could not reach /admin/setup — skipping configured-banner check")
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=3)
        except Exception:
            proc.kill()

    # Third boot — admin user now exists
    out, _ = _spawn_shark(isolated_data_dir, _free_port())
    assert "admin configured" in out.lower() or "sign in with your admin key" in out.lower(), (
        f"expected 'admin configured' banner with admin user present, got:\n{out}"
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
