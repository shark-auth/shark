"""W1.8 Surface 1 — boot output polish.

The bootlog helper provides ✓/✗ phase markers and a Dashboard banner. The
banner prints synchronously before the goroutine starts the listener, so any
spawned shark binary should produce the dashboard line in stdout/server.log.

The conftest fixture writes shark stdout+stderr to server.log; this test
reads that file after boot and asserts the banner text is present.
"""
import os
import re


def test_boot_output_has_dashboard_banner(server):
    """server fixture (in conftest.py) spawns shark serve and waits for /healthz.
    By the time this test runs the boot banner has been printed."""
    log_path = "server.log"
    assert os.path.exists(log_path), f"expected {log_path} from server fixture"
    with open(log_path, encoding="utf-8", errors="replace") as f:
        content = f.read()
    # The banner prints "Dashboard" + the admin URL. ANSI escape codes may be
    # interleaved on TTY, so accept either bare or ANSI-wrapped form.
    assert re.search(r"Dashboard.*http://localhost:\d+/admin", content, re.DOTALL), (
        f"expected dashboard banner in server.log; got first 600 chars:\n{content[:600]}"
    )


def test_boot_output_does_not_crash_silent(server):
    """Negative — boot must not produce a Go stack trace or panic in stdout."""
    log_path = "server.log"
    assert os.path.exists(log_path)
    with open(log_path, encoding="utf-8", errors="replace") as f:
        content = f.read()
    # If shark panicked, the runtime would have written "goroutine 1 [running]"
    # and "panic:" to the log. Their presence means the boot failed mid-flight.
    assert "panic:" not in content, (
        f"shark panicked during boot; log:\n{content[:1200]}"
    )
