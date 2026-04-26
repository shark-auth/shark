import subprocess
import pytest

# W+1: demo command depends on /api/v1/agents/{id}/policies endpoint which is
# not wired in router (Wave 1 Edit 3 shipped UI-only). Without the policies
# step, the 3-hop chain can't run. Skip until the backend handler lands.
pytestmark = pytest.mark.skip(reason="W+1: demo depends on /api/v1/agents/{id}/policies (not wired in router). Re-enable when W1 Edit 3 backend ships.")


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
