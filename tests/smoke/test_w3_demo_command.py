import os
import subprocess
import pytest

_BIN = "./shark.exe" if os.name == "nt" else "./shark"

def test_demo_delegation_with_trace_runs(server, admin_key):
    """The demo command should run end-to-end against the running server in <30s."""
    res = subprocess.run(
        [_BIN, "demo", "delegation-with-trace",
         "--base-url", "http://localhost:8080",
         "--admin-key", admin_key,
         "--plain", "--fast"],
        capture_output=True, text=True, encoding="utf-8", timeout=60,
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
    # Wave M (2026-04-27): hardcoded "DPoP proofs: 3/3 verified" was a lie —
    # replaced with honest claim that all tokens are cnf.jkt-bound.
    assert "cnf.jkt set on all 3 issued tokens" in out
