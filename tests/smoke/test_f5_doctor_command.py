"""
F5 smoke tests — shark doctor command.

Coverage:
  1. All 9 checks are present in human-readable output
  2. Exit code 0 against a fresh-but-configured shark instance
  3. Exit code 1 when admin.key.firstboot is absent AND DB has no api_keys
  4. --json produces parseable JSON per check (one object per line)
  5. Port-in-use scenario produces actionable error mentioning PID or "in use"

The `server` and `admin_key` fixtures from conftest.py bring up a live shark
instance for tests 1, 2, and 4.  Tests 3 and 5 run the binary in isolation
(no running server) against a crafted environment.
"""
import json
import os
import socket
import subprocess
import time
import pytest

BASE_URL = os.environ.get("BASE", "http://localhost:8080")
_shark_db_env = os.environ.get("SHARK_DB_PATH", "")
DB_PATH = _shark_db_env if _shark_db_env else "shark.db"
BIN_PATH = "./shark.exe" if os.name == "nt" else "./shark"

EXPECTED_CHECK_NAMES = [
    "config",
    "db_writability",
    "migrations",
    "jwt_keys",
    "port_bind",
    "base_url",
    "admin_key",
    "smtp",
    "vault",
]


# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------

def run_doctor(*extra_args, env=None, timeout=30):
    """Run `shark doctor [extra_args]` and return (returncode, stdout, stderr)."""
    merged_env = os.environ.copy()
    merged_env["SHARK_DB_PATH"] = DB_PATH
    if env:
        merged_env.update(env)
    result = subprocess.run(
        [BIN_PATH, "doctor"] + list(extra_args),
        capture_output=True,
        text=True,
        timeout=timeout,
        env=merged_env,
    )
    return result.returncode, result.stdout, result.stderr


# ---------------------------------------------------------------------------
# test 1 — all 9 check names appear in human-readable output
# ---------------------------------------------------------------------------

def test_all_nine_checks_present(server, admin_key):
    """All 9 diagnostic check names must appear in the output."""
    rc, stdout, stderr = run_doctor()
    output = stdout + stderr
    for name in EXPECTED_CHECK_NAMES:
        assert name in output, (
            f"Expected check '{name}' not found in doctor output.\n"
            f"stdout:\n{stdout}\nstderr:\n{stderr}"
        )


# ---------------------------------------------------------------------------
# test 2 — exit code 0 against a configured deployment
# ---------------------------------------------------------------------------

def test_exit_code_zero_when_healthy(server, admin_key):
    """shark doctor must exit 0 when shark is fully configured and running."""
    rc, stdout, stderr = run_doctor()
    assert rc == 0, (
        f"Expected exit code 0, got {rc}.\n"
        f"stdout:\n{stdout}\nstderr:\n{stderr}"
    )
    # Final summary line should show all checks passed
    assert "9/9 checks passed" in stdout, (
        f"Expected '9/9 checks passed' in output.\nstdout:\n{stdout}"
    )


# ---------------------------------------------------------------------------
# test 3 — exit code 1 when admin key missing and DB has no api_keys
# ---------------------------------------------------------------------------

def test_exit_code_one_when_admin_key_missing(tmp_path):
    """
    When admin.key.firstboot is absent AND the DB has no api_keys rows,
    shark doctor must exit 1 and report admin_key failure.

    We point SHARK_DB_PATH at a non-existent file so the DB open fails
    (or is empty) and the key file check fails too.
    """
    fake_db = str(tmp_path / "empty.db")
    # Remove any key file that might be in cwd
    for candidate in ["admin.key.firstboot", "data/admin.key.firstboot"]:
        if os.path.exists(candidate):
            try:
                os.rename(candidate, candidate + ".bak_f5test")
            except OSError:
                pass

    try:
        env = {"SHARK_DB_PATH": fake_db}
        rc, stdout, stderr = run_doctor(env=env)
        assert rc == 1, (
            f"Expected exit code 1 (checks failed), got {rc}.\n"
            f"stdout:\n{stdout}\nstderr:\n{stderr}"
        )
        output = stdout + stderr
        assert "admin_key" in output, (
            f"Expected 'admin_key' check in output.\nstdout:\n{stdout}"
        )
        # The check should be marked as failed (✗)
        assert "✗" in output or "false" in output.lower(), (
            f"Expected failure marker for admin_key.\nstdout:\n{stdout}"
        )
    finally:
        # Restore key file backups
        for candidate in ["admin.key.firstboot", "data/admin.key.firstboot"]:
            bak = candidate + ".bak_f5test"
            if os.path.exists(bak):
                try:
                    os.rename(bak, candidate)
                except OSError:
                    pass


# ---------------------------------------------------------------------------
# test 4 — --json emits parseable JSON per check
# ---------------------------------------------------------------------------

def test_json_flag_produces_valid_json(server, admin_key):
    """--json must produce one JSON object per line, each with name/ok/detail."""
    rc, stdout, stderr = run_doctor("--json")
    lines = [ln.strip() for ln in stdout.splitlines() if ln.strip()]
    assert len(lines) >= len(EXPECTED_CHECK_NAMES), (
        f"Expected at least {len(EXPECTED_CHECK_NAMES)} JSON lines, got {len(lines)}.\n"
        f"stdout:\n{stdout}"
    )

    check_objects = []
    for line in lines:
        try:
            obj = json.loads(line)
        except json.JSONDecodeError as e:
            pytest.fail(f"Invalid JSON line: {line!r}\nError: {e}")
        check_objects.append(obj)

    # Collect check objects (those with 'name' and 'ok')
    checks = [o for o in check_objects if "name" in o and "ok" in o]
    check_names_found = {c["name"] for c in checks}

    for expected in EXPECTED_CHECK_NAMES:
        assert expected in check_names_found, (
            f"Check '{expected}' not found in JSON output.\n"
            f"Found checks: {sorted(check_names_found)}"
        )

    # Every check object must have required fields
    for c in checks:
        assert "name" in c, f"Missing 'name' in check object: {c}"
        assert "ok" in c, f"Missing 'ok' in check object: {c}"
        assert "detail" in c, f"Missing 'detail' in check object: {c}"
        assert isinstance(c["ok"], bool), f"'ok' must be bool, got: {c}"

    # Summary line must also be present
    summary_objects = [o for o in check_objects if "summary" in o]
    assert len(summary_objects) == 1, (
        f"Expected exactly one summary JSON object.\nAll objects: {check_objects}"
    )
    summary = summary_objects[0]
    assert summary["passed"] == summary["total"], (
        f"Expected all checks to pass, got: {summary}"
    )


# ---------------------------------------------------------------------------
# test 5 — port-in-use reports PID or actionable message
# ---------------------------------------------------------------------------

def test_port_in_use_reports_actionable_error():
    """
    When the configured port is already bound, doctor must report an actionable
    error that at least mentions the port number. Best-effort: if PID detection
    works on this platform, it must mention 'PID'.
    """
    # Bind the port ourselves to simulate EADDRINUSE
    port = 18099  # use a non-default port to avoid clashing with a real server

    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    try:
        sock.bind(("", port))
        sock.listen(1)

        env = {
            "SHARKAUTH_SERVER__PORT": str(port),
            "SHARK_DB_PATH": DB_PATH,
        }
        rc, stdout, stderr = run_doctor(env=env)
        output = stdout + stderr

        # port_bind check must appear
        assert "port_bind" in output, (
            f"Expected 'port_bind' check in output.\nstdout:\n{stdout}"
        )

        # Must mention the port number
        assert str(port) in output, (
            f"Expected port {port} mentioned in output.\nstdout:\n{stdout}"
        )

        # Must contain an actionable signal — either "in use", "PID", or process info
        actionable = any(
            kw in output.lower()
            for kw in ["in use", "pid", "already", "eaddrinuse"]
        )
        assert actionable, (
            f"Expected actionable error (in use / PID) for port {port}.\n"
            f"stdout:\n{stdout}\nstderr:\n{stderr}"
        )

        # Exit code must be 1 (at least port_bind failed)
        assert rc == 1, (
            f"Expected exit code 1 when port is in use, got {rc}.\n"
            f"stdout:\n{stdout}"
        )
    finally:
        sock.close()
