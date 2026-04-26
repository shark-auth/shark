"""
Wave 3 smoke — vault hop step is printed on stdout when demo runs.

The demo command requires a running shark server + admin key.
In CI these are provided via SHARK_BASE_URL / SHARK_ADMIN_KEY env vars.
If not set the test is skipped so it never blocks the base smoke suite.
"""
import os
import subprocess
import sys

import pytest


SHARK_EXE = os.path.join(os.path.dirname(__file__), "shark.exe")
BASE_URL = os.environ.get("SHARK_BASE_URL", "")
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")


@pytest.mark.skipif(
    not BASE_URL or not ADMIN_KEY,
    reason="SHARK_BASE_URL and SHARK_ADMIN_KEY required for demo smoke tests",
)
def test_demo_vault_hop_exit_zero():
    """shark demo delegation-with-trace exits 0 and prints vault hop line."""
    report = "./demo-report-smoke.html"
    result = subprocess.run(
        [
            SHARK_EXE,
            "demo",
            "delegation-with-trace",
            "--base-url", BASE_URL,
            "--admin-key", ADMIN_KEY,
            "--output", report,
            "--no-open",
        ],
        capture_output=True,
        text=True,
        timeout=60,
        check=False,
    )

    assert result.returncode == 0, (
        f"demo command exited {result.returncode}\n"
        f"stdout: {result.stdout}\n"
        f"stderr: {result.stderr}"
    )

    combined = result.stdout + result.stderr

    # Vault hop print line must appear
    assert "[4/4]" in combined, f"Missing [4/4] vault hop line in output:\n{combined}"
    assert "google_gmail" in combined, f"Missing 'google_gmail' in output:\n{combined}"

    # Report file must exist
    assert os.path.exists(report), f"Report file not created at {report}"

    content = open(report, encoding="utf-8").read()
    assert "google_gmail" in content, "google_gmail not found in HTML report"

    # Cleanup
    try:
        os.remove(report)
    except OSError:
        pass


@pytest.mark.skipif(
    not BASE_URL or not ADMIN_KEY,
    reason="SHARK_BASE_URL and SHARK_ADMIN_KEY required for demo smoke tests",
)
def test_demo_vault_hop_stdout_format():
    """Verify the 4-step trace lines appear in correct order."""
    result = subprocess.run(
        [
            SHARK_EXE,
            "demo",
            "delegation-with-trace",
            "--base-url", BASE_URL,
            "--admin-key", ADMIN_KEY,
            "--no-open",
        ],
        capture_output=True,
        text=True,
        timeout=60,
        check=False,
    )

    stdout = result.stdout
    assert "[1/3]" in stdout
    assert "[2/3]" in stdout
    assert "[3/3]" in stdout
