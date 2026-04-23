import pytest
import subprocess
import shutil
import os
import requests

BASE_URL = os.environ.get("BASE", "http://localhost:8080")

@pytest.mark.skipif(not shutil.which("pnpm"), reason="pnpm not found")
def test_sdk_react_build():
    """Section 76: SDK integration (@shark-auth/react package build)."""
    # Use shell=True for windows compatibility with pnpm
    res = subprocess.run("pnpm --filter @shark-auth-react build", shell=True, capture_output=True, text=True)
    assert res.returncode == 0, f"SDK build failed: {res.stderr}"

@pytest.mark.skipif(not shutil.which("pnpm"), reason="pnpm not found")
def test_sdk_react_test():
    """Section 76: SDK integration (@shark-auth/react tests)."""
    res = subprocess.run("pnpm --filter @shark-auth-react test:run", shell=True, capture_output=True, text=True)
    # Note: Using test:run as per smoke_test.sh pattern or similar
    if res.returncode != 0:
        # Fallback if command differs
        res = subprocess.run("pnpm --filter @shark-auth-react test", shell=True, capture_output=True, text=True)
    assert res.returncode == 0, f"SDK tests failed: {res.stderr}"

def test_snippet_endpoint(admin_client):
    """Section 76: Snippet endpoint verification."""
    # 1. Get first app
    apps = admin_client.get(f"{BASE_URL}/api/v1/admin/apps").json().get("data", [])
    if not apps:
        pytest.skip("No apps found to test snippet endpoint")
    
    app_id = apps[0]["id"]
    
    # 2. GET React snippets
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/apps/{app_id}/snippet?framework=react")
    assert resp.status_code == 200
    assert len(resp.json()["snippets"]) == 3
    
    # 3. Unsupported framework
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/apps/{app_id}/snippet?framework=vue")
    assert resp.status_code == 501
    assert resp.json()["error"] == "framework_not_supported"
