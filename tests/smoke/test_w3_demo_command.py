import subprocess
import pytest

def test_demo_delegation_with_trace_runs(server, admin_key):
    """The demo command should run end-to-end against the running server in <30s."""
    res = subprocess.run(
        ["./shark.exe", "demo", "delegation-with-trace",
         "--base-url", "http://localhost:8080",
         "--admin-key", admin_key,
         "--plain"],
        capture_output=True, text=True, timeout=60,
    )
    assert res.returncode == 0, f"stdout: {res.stdout}\nstderr: {res.stderr}"
    out = res.stdout
    assert "[1/3] Registering agents" in out
    assert "[2/3] Configuring may_act" in out
    assert "[3/3] Running delegation chain" in out
    assert "user → user-proxy → email-service → followup-service" in out
    assert "Token 1:" in out
    assert "Token 2:" in out
    assert "Token 3:" in out
    assert "DPoP proofs: 3/3 verified" in out
