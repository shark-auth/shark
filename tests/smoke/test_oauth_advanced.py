import pytest
import requests
import base64
import hashlib
import json
import time

import os
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_pkce_enforcement(admin_client, auth_session):
    """Section 32: PKCE enforcement (OAuth 2.1)."""
    agent = admin_client.post(f"{BASE_URL}/api/v1/agents", json={
        "name": "pkce-enforce",
        "grant_types": ["authorization_code"],
        "scopes": ["openid"]
    }).json()
    
    qs_params = {
        "response_type": "code",
        "client_id": agent['client_id'],
        "redirect_uri": "http://localhost:9999/callback",
        "state": "noPkce",
        "scope": "openid"
    }
    resp = auth_session.get(f"{BASE_URL}/oauth/authorize", params=qs_params, allow_redirects=False)
    
    if resp.status_code == 302:
        loc = resp.headers.get("Location", "")
        assert "error=" in loc or "error_description=" in loc
    else:
        assert resp.status_code in [400, 401]

def test_refresh_token_rotation_and_reuse(admin_client, smoke_user):
    """Section 33: Refresh token rotation and reuse protection.

    Uses a dedicated session with its own cookie jar so the authorize
    redirect carries the correct login cookie independent of any other
    test that might share auth_session's cookie jar.
    """
    import urllib.parse

    # 1. Create agent with offline_access scope allowed.
    agent_resp = admin_client.post(f"{BASE_URL}/api/v1/agents", json={
        "name": "rt-rotation",
        "grant_types": ["authorization_code", "refresh_token"],
        "scopes": ["openid", "offline_access"],
        "redirect_uris": ["http://localhost:9999/callback"]
    })
    assert agent_resp.status_code in [200, 201], f"Agent create failed: {agent_resp.text}"
    agent = agent_resp.json()
    assert "client_id" in agent, f"No client_id in agent response: {agent}"
    cid, secret = agent["client_id"], agent["client_secret"]

    # 2. Fresh session — login to get a valid session cookie.
    sess = requests.Session()
    login = sess.post(f"{BASE_URL}/api/v1/auth/login",
                      json={"email": smoke_user["email"], "password": smoke_user["password"]})
    assert login.status_code == 200, f"Login failed: {login.text}"

    # 3. Complete Code Flow with PKCE to get a refresh token.
    verifier = "v" * 64
    challenge = base64.urlsafe_b64encode(
        hashlib.sha256(verifier.encode()).digest()
    ).decode().rstrip('=')
    qs_params = {
        "response_type": "code",
        "client_id": cid,
        "redirect_uri": "http://localhost:9999/callback",
        "state": "state12345",
        "code_challenge": challenge,
        "code_challenge_method": "S256",
        "scope": "openid offline_access",
    }

    resp = sess.get(f"{BASE_URL}/oauth/authorize", params=qs_params, allow_redirects=False)
    if resp.status_code == 200:
        # Consent form — POST approval.
        qs = urllib.parse.urlencode(qs_params)
        resp = sess.post(
            f"{BASE_URL}/oauth/authorize?{qs}",
            data={"client_id": cid, "state": "state12345",
                  "approved": "true", "scope": "openid offline_access"},
            allow_redirects=False,
        )

    assert "Location" in resp.headers, (
        f"Expected redirect from authorize, got {resp.status_code}: {resp.text}"
    )
    loc = resp.headers["Location"]
    parsed = requests.utils.urlparse(loc)
    qs_dict = dict(urllib.parse.parse_qsl(parsed.query))
    assert "code" in qs_dict, f"No code in redirect: {loc}"
    code = qs_dict["code"]

    # 4. Exchange code for tokens.
    # Server registers confidential clients with token_endpoint_auth_method=
    # client_secret_basic — send credentials via HTTP Basic, not form body.
    token_resp = requests.post(f"{BASE_URL}/oauth/token",
        auth=(cid, secret),
        data={
            "grant_type": "authorization_code",
            "code": code,
            "redirect_uri": "http://localhost:9999/callback",
            "code_verifier": verifier,
        },
    )
    assert token_resp.status_code == 200, f"Token exchange failed: {token_resp.text}"
    tokens = token_resp.json()
    assert "refresh_token" in tokens, f"No refresh_token in response: {tokens}"
    rt1 = tokens["refresh_token"]

    # 5. First Refresh — fosite MUST issue a new RT (rotation).
    resp = requests.post(f"{BASE_URL}/oauth/token",
        auth=(cid, secret),
        data={
            "grant_type": "refresh_token",
            "refresh_token": rt1,
        },
    )
    assert resp.status_code == 200, f"First refresh failed: {resp.text}"
    body = resp.json()
    assert "refresh_token" in body, f"No refresh_token after rotation: {body}"
    rt2 = body["refresh_token"]
    assert rt1 != rt2, "Refresh token was NOT rotated (rt1 == rt2)"

    # 6. Reuse rt1 — MUST be rejected (family breach detection).
    resp = requests.post(f"{BASE_URL}/oauth/token",
        auth=(cid, secret),
        data={
            "grant_type": "refresh_token",
            "refresh_token": rt1,
        },
    )
    assert resp.status_code in [400, 401], (
        f"Old refresh token still accepted after rotation (expected 400/401, got {resp.status_code})"
    )

def test_device_flow(admin_client, smoke_user, db_conn):
    """Section 34: Device flow (RFC 8628)."""
    # 1. Create Device Agent
    agent = admin_client.post(f"{BASE_URL}/api/v1/agents", json={
        "name": "device-agent",
        "grant_types": ["urn:ietf:params:oauth:grant-type:device_code"],
        "scopes": ["read"]
    }).json()
    cid = agent["client_id"]
    
    # 2. Request Device Code
    resp = requests.post(f"{BASE_URL}/oauth/device", data={"client_id": cid, "scope": "read"})
    assert resp.status_code == 200
    data = resp.json()
    device_code = data["device_code"]
    user_code = data["user_code"]
    
    # 3. Poll (Pending)
    resp = requests.post(f"{BASE_URL}/oauth/token", data={
        "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
        "device_code": device_code, "client_id": cid
    })
    assert resp.status_code == 400
    assert resp.json()["error"] == "authorization_pending"
    
    # 4. Approve via DB
    cursor = db_conn.cursor()
    cursor.execute("UPDATE oauth_device_codes SET status='approved', user_id=? WHERE user_code=?", (smoke_user["id"], user_code))
    db_conn.commit()
    
    # 5. Poll (Success) with backoff
    for _ in range(10):
        resp = requests.post(f"{BASE_URL}/oauth/token", data={
            "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
            "device_code": device_code, "client_id": cid
        })
        if resp.status_code == 200:
            break
        if resp.status_code == 400 and resp.json().get("error") == "slow_down":
            time.sleep(5)
        else:
            time.sleep(1)
        
    assert resp.status_code == 200, f"Device flow poll failed: {resp.text}"
    assert "access_token" in resp.json()

def test_token_exchange(admin_client):
    """Section 35: Token Exchange (RFC 8693)."""
    agent = admin_client.post(f"{BASE_URL}/api/v1/agents", json={
        "name": "exchange-agent",
        "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
        "scopes": ["read"]
    }).json()
    cid, secret = agent["client_id"], agent["client_secret"]
    
    tok = requests.post(f"{BASE_URL}/oauth/token", auth=(cid, secret), data={"grant_type": "client_credentials"}).json()["access_token"]
    
    resp = requests.post(f"{BASE_URL}/oauth/token", auth=(cid, secret), data={
        "grant_type": "urn:ietf:params:oauth:grant-type:token-exchange",
        "subject_token": tok,
        "subject_token_type": "urn:ietf:params:oauth:token-type:access_token"
    })
    assert resp.status_code in [200, 400]

def test_dpop_surface(admin_client):
    """Section 36: DPoP (RFC 9449) Surface."""
    agent = admin_client.post(f"{BASE_URL}/api/v1/agents", json={"name":"dpop","grant_types":["client_credentials"]}).json()
    cid, secret = agent["client_id"], agent["client_secret"]
    
    resp = requests.post(f"{BASE_URL}/oauth/token", auth=(cid, secret), headers={"DPoP": "garbage"}, data={"grant_type": "client_credentials"})
    assert resp.status_code in [400, 401]

def test_introspection_revocation(admin_client):
    """Section 37 & 38: Introspection and OAuth Revocation."""
    agent = admin_client.post(f"{BASE_URL}/api/v1/agents", json={"name":"ir","grant_types":["client_credentials"]}).json()
    cid, secret = agent["client_id"], agent["client_secret"]
    
    tok = requests.post(f"{BASE_URL}/oauth/token", auth=(cid, secret), data={"grant_type": "client_credentials"}).json()["access_token"]
    
    resp = requests.post(f"{BASE_URL}/oauth/introspect", auth=(cid, secret), data={"token": tok})
    assert resp.json()["active"] is True
    
    requests.post(f"{BASE_URL}/oauth/revoke", auth=(cid, secret), data={"token": tok})
    
    resp = requests.post(f"{BASE_URL}/oauth/introspect", auth=(cid, secret), data={"token": tok})
    assert resp.json()["active"] is False

def test_dcr_lifecycle(admin_client):
    """Section 39: Dynamic Client Registration (RFC 7591)."""
    resp = requests.post(f"{BASE_URL}/oauth/register", json={
        "client_name": "dcr-test", "grant_types": ["client_credentials"], "scope": "read"
    })
    assert resp.status_code == 201
    data = resp.json()
    cid, rat = data["client_id"], data["registration_access_token"]
    
    resp = requests.get(f"{BASE_URL}/oauth/register/{cid}", headers={"Authorization": f"Bearer {rat}"})
    assert resp.status_code == 200
    assert resp.json()["client_name"] == "dcr-test"
    
    resp = requests.put(f"{BASE_URL}/oauth/register/{cid}", headers={"Authorization": f"Bearer {rat}"}, json={
        "client_name": "dcr-updated", "grant_types": ["client_credentials"]
    })
    assert resp.status_code == 200
    
    resp = requests.delete(f"{BASE_URL}/oauth/register/{cid}", headers={"Authorization": f"Bearer {rat}"})
    assert resp.status_code == 204

def test_resource_indicators(admin_client):
    """Section 40: Resource Indicators (RFC 8707)."""
    agent = admin_client.post(f"{BASE_URL}/api/v1/agents", json={"name":"res","grant_types":["client_credentials"]}).json()
    cid, secret = agent["client_id"], agent["client_secret"]
    
    resp = requests.post(f"{BASE_URL}/oauth/token", auth=(cid, secret), data={
        "grant_type": "client_credentials",
        "resource": "https://api.example.com"
    })
    tok = resp.json()["access_token"]
    
    intro = requests.post(f"{BASE_URL}/oauth/introspect", auth=(cid, secret), data={"token": tok}).json()
    assert intro["aud"] == "https://api.example.com"

def test_jwks_es256(api_session):
    """Section 41: ES256 JWKS validation."""
    resp = requests.get(f"{BASE_URL}/.well-known/jwks.json")
    jwks = resp.json()
    es256_keys = [k for k in jwks["keys"] if k.get("alg") == "ES256"]
    assert len(es256_keys) > 0
    assert es256_keys[0]["kty"] == "EC"
    assert es256_keys[0]["crv"] == "P-256"
