import pytest
import requests
import time

import os
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_rbac_reverse_lookup(admin_client, smoke_user):
    """Section 66: RBAC reverse lookup + email preview."""
    perm_name = f"perm_{int(time.time()*1000)}"
    perm = admin_client.post(f"{BASE_URL}/api/v1/permissions", json={"action": perm_name, "resource": "thing"}).json()
    role = admin_client.post(f"{BASE_URL}/api/v1/roles", json={"name": f"role_{int(time.time()*1000)}"}).json()
    
    admin_client.post(f"{BASE_URL}/api/v1/roles/{role['id']}/permissions", json={"permission_id": perm['id']})
    admin_client.post(f"{BASE_URL}/api/v1/users/{smoke_user['id']}/roles", json={"role_id": role['id']})
    
    roles_data = admin_client.get(f"{BASE_URL}/api/v1/permissions/{perm['id']}/roles").json()
    roles = roles_data.get("data", roles_data.get("roles", []))
    assert any(r["id"] == role["id"] for r in roles)
    
    users_data = admin_client.get(f"{BASE_URL}/api/v1/permissions/{perm['id']}/users").json()
    users = users_data.get("data", users_data.get("users", []))
    assert any(u["id"] == smoke_user["id"] for u in users)

def test_email_previews(admin_client):
    """Section 66 (part 2): Email preview rendering."""
    for tpl in ["magic_link", "verify_email", "password_reset", "organization_invitation"]:
        resp = admin_client.get(f"{BASE_URL}/api/v1/admin/email-preview/{tpl}")
        assert resp.status_code == 200
        data = resp.json()
        assert "html" in data
        assert len(data["html"]) > 100

def test_proxy_rules_crud(admin_client):
    """Section 67 & 68: Proxy rules CRUD (DB-backed)."""
    # 1. Create App first (required by modern backend)
    app = admin_client.post(f"{BASE_URL}/api/v1/admin/apps", json={"name": "ProxyTestApp"}).json()
    app_id = app["id"]

    # 2. Create Proxy Rule
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/proxy/rules/db", json={
        "app_id": app_id,
        "name": f"Smoke Override {time.time()}",
        "pattern": "/api/smoke/{id}",
        "methods": ["GET", "PATCH"],
        "require": "role:admin",
        "scopes": ["webhooks:write"],
        "enabled": True,
        "priority": 50
    })
    assert resp.status_code == 201, f"Failed to create proxy rule: {resp.text}"
    body = resp.json()
    rule = body.get("data", body)
    rule_id = rule["id"]
    
    admin_client.patch(f"{BASE_URL}/api/v1/admin/proxy/rules/db/{rule_id}", json={"enabled": False})
    
    resp = admin_client.get(f"{BASE_URL}/api/v1/admin/proxy/rules/db/{rule_id}")
    rule_get = resp.json().get("data", resp.json())
    assert rule_get["enabled"] is False
    
    admin_client.delete(f"{BASE_URL}/api/v1/admin/proxy/rules/db/{rule_id}")

def test_audit_export_csv(admin_client):
    """Section 69: Audit log CSV export."""
    payload = {
        "from": "2020-01-01T00:00:00Z",
        "to": "2030-01-01T00:00:00Z"
    }
    # Ensure there is at least one audit log
    admin_client.get(f"{BASE_URL}/api/v1/admin/stats")
    
    resp = admin_client.post(f"{BASE_URL}/api/v1/audit-logs/export", json=payload)
    if resp.status_code == 200:
        assert "text/csv" in resp.headers["Content-Type"]
    else:
        # Fallback if no matching logs
        assert resp.status_code in [200, 400]

def test_admin_user_creation(admin_client):
    """Section 70: POST /admin/users admin-key user creation."""
    email = f"admin-user-{int(time.time()*1000)}@test.com"
    resp = admin_client.post(f"{BASE_URL}/api/v1/admin/users", json={
        "email": email, "password": "Password123!", "name": "T04 Admin"
    })
    assert resp.status_code == 201
    assert resp.json()["email"] == email
