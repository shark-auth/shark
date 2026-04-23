import pytest
import requests

import os
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

@pytest.fixture
def auth_session(api_session):
    """Reuse the session from signup if available."""
    user = getattr(pytest, "smoke_user", None)
    if not user:
        # Fallback signup
        email = f"orgsmoke@test.com"
        api_session.post(f"{BASE_URL}/api/v1/auth/signup", json={"email": email, "password": "Password123!"})
    return api_session

def test_organization_rbac(auth_session):
    """Section 11: Organization creation and RBAC roles."""
    # 1. Create Org
    payload = {
        "name": "Acme",
        "slug": f"acme-smoke-{requests.utils.quote('test')}" # Just slightly unique
    }
    resp = auth_session.post(f"{BASE_URL}/api/v1/organizations", json=payload)
    assert resp.status_code == 201, resp.text
    org_id = resp.json()["id"]
    
    # 2. List Roles (expect 3 builtin)
    resp = auth_session.get(f"{BASE_URL}/api/v1/organizations/{org_id}/roles")
    assert resp.status_code == 200
    roles = resp.json()["data"]
    assert len(roles) == 3, f"Expected 3 builtin roles, got {len(roles)}"
    
    # 3. Create Custom Role
    resp = auth_session.post(f"{BASE_URL}/api/v1/organizations/{org_id}/roles", json={
        "name": "editor",
        "description": "custom editor"
    })
    assert resp.status_code == 201
    role_id = resp.json()["id"]
    
    # 4. Try delete builtin (should be 409)
    owner_role = next(r for r in roles if r["name"] == "owner")
    resp = auth_session.delete(f"{BASE_URL}/api/v1/organizations/{org_id}/roles/{owner_role['id']}")
    assert resp.status_code == 409
    
    # Clean up custom role
    auth_session.delete(f"{BASE_URL}/api/v1/organizations/{org_id}/roles/{role_id}")
