import pytest
import subprocess
import os
import requests
import time

BIN_PATH = "./shark.exe" if os.name == 'nt' else "./shark"
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_key_rotation_cli(server, admin_key):
    """Section 7: Key rotation via CLI (W17: uses admin API, no server restart)."""
    # W17: keys rotate via admin API — do NOT terminate the session-scoped server.
    res = subprocess.run(
        [BIN_PATH, "keys", "generate-jwt", "--url", BASE_URL, "--token", admin_key],
        capture_output=True, text=True,
    )
    assert res.returncode == 0, f"keys generate-jwt failed: {res.stderr}"

    # Verify JWKS has keys (rotation keeps old + new keys)
    resp = requests.get(f"{BASE_URL}/.well-known/jwks.json")
    keys = resp.json().get("keys", [])
    assert len(keys) >= 1, f"Expected >=1 key in JWKS after rotation, got {len(keys)}"

def test_apps_cli(server, admin_key):
    """Section 8: Apps CLI management."""
    import re
    # Create app
    res = subprocess.run(
        [BIN_PATH, "app", "create", "--name", "cliapp", "--callback", "https://ok.com",
         "--url", BASE_URL, "--token", admin_key],
        capture_output=True, text=True,
    )
    assert res.returncode == 0, f"app create failed: {res.stderr}"
    assert "client_id" in res.stdout
    cid = re.search(r'shark_app_[A-Za-z0-9_-]+', res.stdout).group(0)

    # List
    res = subprocess.run(
        [BIN_PATH, "app", "list", "--url", BASE_URL, "--token", admin_key],
        capture_output=True, text=True,
    )
    assert cid in res.stdout

    # Show (security check)
    res = subprocess.run(
        [BIN_PATH, "app", "show", cid, "--url", BASE_URL, "--token", admin_key],
        capture_output=True, text=True,
    )
    assert "client_secret_hash" not in res.stdout, "Secret hash leaked in CLI show"

    # Rotate secret
    res = subprocess.run(
        [BIN_PATH, "app", "rotate-secret", cid, "--url", BASE_URL, "--token", admin_key],
        capture_output=True, text=True,
    )
    assert "client_secret" in res.stdout

def test_admin_system_endpoints(admin_client):
    """Section 14: Admin System Endpoints."""
    # Test email
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/test-email", json={"to": "test@example.com"})
    assert resp.status_code == 200
    
    # Purge sessions
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/sessions/purge-expired")
    assert resp.status_code == 200
    
    # Purge audit
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/audit-logs/purge", json={"before": "2020-01-01T00:00:00Z"})
    assert resp.status_code == 200
    
    # Rotate signing key via API (Section 14 part)
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/auth/rotate-signing-key")
    assert resp.status_code == 200
