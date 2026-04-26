import pytest
import requests
import time

import os
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_consent_management(auth_session, admin_client, smoke_user):
    """Section 42: Consent management self-service and admin list."""
    # 1. User view
    resp = auth_session.get(f"{BASE_URL}/api/v1/auth/consents")
    assert resp.status_code == 200
    assert "data" in resp.json()
    
    # 2. Admin view
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/oauth/consents")
    assert resp.status_code == 200

def test_vault_and_templates(admin_client):
    """Section 43 & 44: Vault providers and templates."""
    # Templates discovery
    resp = admin_client.get(f"{BASE_URL}/api/v1/vault/templates")
    assert resp.status_code == 200
    templates = resp.json()["data"]
    assert any(t["name"] == "github" for t in templates)
    
    # Provider CRUD
    resp = admin_client.post(f"{BASE_URL}/api/v1/vault/providers", json={
        "template": "github",
        "client_id": "smoke-id",
        "client_secret": "smoke-secret"
    })
    assert resp.status_code == 201
    vp_id = resp.json()["id"]
    
    # Update
    admin_client.patch(f"{BASE_URL}/api/v1/vault/providers/{vp_id}", json={"display_name": "Updated Vault"})
    
    # Delete
    resp = admin_client.delete(f"{BASE_URL}/api/v1/vault/providers/{vp_id}")
    assert resp.status_code == 204

def test_vault_connect_flow(auth_session, admin_client):
    """Section 45: Vault connect flow (session auth)."""
    # Create seed provider
    admin_client.post(f"{BASE_URL}/api/v1/vault/providers", json={
        "template": "github", "client_id": "c1", "client_secret": "s1"
    })
    
    # Connect (Expect 302 to upstream)
    resp = auth_session.get(f"{BASE_URL}/api/v1/vault/connect/github", allow_redirects=False)
    assert resp.status_code == 302
    assert "github.com" in resp.headers["Location"]
    assert "shark_vault_state" in auth_session.cookies

def test_auth_flow_lifecycle(admin_client):
    """Sections 50-54: Auth flow CRUD, Dry-run and Persistence."""
    # 1. Create Flow
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/flows", json={
        "name": "Smoke Flow",
        "trigger": "signup",
        "steps": [{"type": "require_email_verification"}],
        "enabled": True,
        "priority": 10
    })
    assert resp.status_code == 201
    flow_id = resp.json()["id"]
    
    # 2. Dry-run
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/flows/{flow_id}/test", json={
        "user": {"email": "dry@test.com", "email_verified": False}
    })
    assert resp.json()["outcome"] == "block"
    
    # 3. List runs
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/flows/{flow_id}/runs")
    assert resp.status_code == 200
    
    # 4. Cleanup
    admin_client.delete(f"{BASE_URL}/api/v1/admin/flows/{flow_id}")

def test_flow_blocks_signup(admin_client):
    """Section 52: Flow blocks signup on unverified email."""
    # 1. Create blocking flow
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/flows", json={
        "name": "Blocker", "trigger": "signup",
        "steps": [{"type": "require_email_verification"}], "enabled": True
    })
    flow_id = resp.json()["id"]
    
    # 2. Try signup
    resp = requests.post(f"{BASE_URL}/api/v1/auth/signup", json={
        "email": f"blocked{time.time()}@test.com", "password": "Password123!"
    })
    assert resp.status_code == 403
    assert resp.json()["error"] == "flow_blocked"
    
    # 3. Cleanup
    admin_client.delete(f"{BASE_URL}/api/v1/admin/flows/{flow_id}")
