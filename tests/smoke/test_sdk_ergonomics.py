import os
import sys
import pytest
import requests
import time

# Add the SDK source to sys.path
sys.path.append(os.path.join(os.path.dirname(__file__), "../../sdk/python"))

import shark_auth
from shark_auth import DPoPProver, AgentSession, exchange_token, decode_agent_token

BASE_URL = os.environ.get("BASE", "http://localhost:8080")

@pytest.fixture
def agent(admin_client):
    """Create a fresh agent for SDK testing."""
    name = f"sdk-agent-{int(time.time())}"
    resp = admin_client.post(f"{BASE_URL}/api/v1/agents", json={
        "name": name,
        "grant_types": ["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"],
        "scopes": ["read", "write"]
    })
    assert resp.status_code == 201
    return resp.json()

def test_agent_session_auto_signing(agent):
    """Test Lane 1: AgentSession auto-signs requests with DPoP."""
    cid, secret = agent["client_id"], agent["client_secret"]
    prover = DPoPProver.generate()
    
    # Get a DPoP-bound token
    token_url = f"{BASE_URL}/oauth/token"
    proof = prover.make_proof("POST", token_url)
    resp = requests.post(token_url, auth=(cid, secret), data={
        "grant_type": "client_credentials"
    }, headers={"DPoP": proof})
    
    assert resp.status_code == 200
    access_token = resp.json()["access_token"]
    
    # Use AgentSession
    session = AgentSession(prover, access_token)
    
    # Call /api/v1/auth/me (or any protected route)
    # The proxy or server should accept the DPoP header
    resp = session.get(f"{BASE_URL}/api/v1/auth/me")
    
    # If the server is in session mode, /auth/me might expect a cookie,
    # but for an agent, it should check the Authorization header.
    # Even if it returns 401, we want to verify the request HEADERS reached the server.
    assert "Authorization" in resp.request.headers
    assert resp.request.headers["Authorization"].startswith("DPoP ")
    assert "DPoP" in resp.request.headers

def test_token_exchange_sdk(agent):
    """Test Lane 3: exchange_token high-level API."""
    cid, secret = agent["client_id"], agent["client_secret"]
    
    # 1. Get initial token with enough scope
    resp = requests.post(f"{BASE_URL}/oauth/token", auth=(cid, secret), data={
        "grant_type": "client_credentials",
        "scope": "read write"
    })
    subject_token = resp.json()["access_token"]
    
    # 2. Use SDK to exchange
    tokens = exchange_token(
        auth_url=BASE_URL,
        client_id=cid,
        client_secret=secret,
        subject_token=subject_token,
        scope="read"
    )
    
    assert tokens.access_token is not None
    assert tokens.token_type.lower() == "bearer"

def test_secure_decode_default(agent):
    """Test Lane 3: decode_agent_token verifies by default."""
    cid, secret = agent["client_id"], agent["client_secret"]
    
    # Get a token
    resp = requests.post(f"{BASE_URL}/oauth/token", auth=(cid, secret), data={
        "grant_type": "client_credentials"
    })
    access_token = resp.json()["access_token"]
    
    # Verify successfully
    # Note: For CC tokens, SharkAuth currently sets the client_id as the audience
    claims = decode_agent_token(
        token=access_token,
        jwks_url=f"{BASE_URL}/.well-known/jwks.json",
        expected_issuer=BASE_URL,
        expected_audience=cid
    )
    
    assert claims.sub == cid
    
    # Test failure: wrong audience
    with pytest.raises(shark_auth.TokenError) as exc:
        decode_agent_token(
            token=access_token,
            jwks_url=f"{BASE_URL}/.well-known/jwks.json",
            expected_issuer=BASE_URL,
            expected_audience="wrong-aud"
        )
    assert "invalid audience" in str(exc.value)
