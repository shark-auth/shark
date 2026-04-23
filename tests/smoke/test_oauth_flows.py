import pytest
import requests
import base64
import hashlib
import re

import os
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_agent_crud(admin_client):
    """Section 29: Agent CRUD via Admin API."""
    payload = {
        "name": "smoke-agent",
        "grant_types": ["client_credentials"],
        "scopes": ["read", "write"]
    }
    resp = admin_client.post(f"{BASE_URL}/api/v1/agents", json=payload)
    assert resp.status_code == 201
    
    agent = resp.json()
    agent_id = agent["id"]
    client_id = agent["client_id"]
    client_secret = agent["client_secret"]
    
    assert client_id.startswith("shark_agent_")
    assert client_secret

    # List
    resp = admin_client.get(f"{BASE_URL}/api/v1/agents")
    assert resp.status_code == 200
    assert any(a["id"] == agent_id for a in resp.json()["data"])
    
    # Patch
    resp = admin_client.patch(f"{BASE_URL}/api/v1/agents/{agent_id}", json={"description": "updated"})
    assert resp.status_code == 200
    assert resp.json()["description"] == "updated"
    
    # Audit
    resp = admin_client.get(f"{BASE_URL}/api/v1/agents/{agent_id}/audit")
    assert resp.status_code == 200
    assert len(resp.json()["data"]) >= 1

    return agent # For CC test

def test_client_credentials_grant(admin_client):
    """Section 30: Client Credentials Flow."""
    # Create CC agent
    agent_resp = admin_client.post(f"{BASE_URL}/api/v1/agents", json={
        "name": "cc-agent",
        "grant_types": ["client_credentials"],
        "scopes": ["read"]
    })
    agent = agent_resp.json()
    
    # Token request
    auth_str = f"{agent['client_id']}:{agent['client_secret']}"
    auth_b64 = base64.b64encode(auth_str.encode()).decode()
    
    resp = requests.post(f"{BASE_URL}/oauth/token", 
        headers={"Authorization": f"Basic {auth_b64}"},
        data={"grant_type": "client_credentials", "scope": "read"}
    )
    assert resp.status_code == 200
    data = resp.json()
    assert "access_token" in data
    assert data["token_type"] in ["Bearer", "bearer", "DPoP"]

def test_auth_code_pkce_flow(admin_client, auth_session):
    """Section 31: Auth Code + PKCE Flow."""
    # 1. Create PKCE Agent
    agent_resp = admin_client.post(f"{BASE_URL}/api/v1/agents", json={
        "name": "pkce-agent",
        "grant_types": ["authorization_code", "refresh_token"],
        "redirect_uris": ["http://localhost:9999/callback"],
        "scopes": ["openid", "profile", "offline_access"],
        "client_type": "confidential",
        "response_types": ["code"]
    })
    agent = agent_resp.json()
    
    # 2. PKCE Prep
    verifier = "test_verifier_0123456789abc_0123456789abc_0123456789"
    challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).decode().rstrip('=')
    
    # 3. Authorize (logged in user required)
    # auth_session fixture already ensures logged in.
    # But for PKCE, we might want a fresh session or just use the cookies.
    
    qs = {
        "response_type": "code",
        "client_id": agent["client_id"],
        "redirect_uri": "http://localhost:9999/callback",
        "state": "xyzabcde",
        "code_challenge": challenge,
        "code_challenge_method": "S256",
        "scope": "openid profile offline_access"
    }
    
    # GET Authorize
    resp = auth_session.get(f"{BASE_URL}/oauth/authorize", params=qs, allow_redirects=False)
    
    # Might be 200 (consent) or 302 (auto-approve)
    if resp.status_code == 200:
        # Post consent
        resp = auth_session.post(f"{BASE_URL}/oauth/authorize", data={
            "challenge": requests.compat.urlencode(qs),
            "client_id": agent["client_id"],
            "state": "xyzabcde",
            "approved": "true"
        }, allow_redirects=False)

    assert resp.status_code in [302, 303]
    loc = resp.headers["Location"]
    code = re.search(r'code=([^&]+)', loc).group(1)
    
    # 4. Exchange
    auth_str = f"{agent['client_id']}:{agent['client_secret']}"
    auth_b64 = base64.b64encode(auth_str.encode()).decode()
    
    resp = requests.post(f"{BASE_URL}/oauth/token",
        headers={"Authorization": f"Basic {auth_b64}"},
        data={
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": "http://localhost:9999/callback",
            "code_verifier": verifier
        }
    )
    
    assert resp.status_code == 200
    data = resp.json()
    assert "access_token" in data
    assert "refresh_token" in data
