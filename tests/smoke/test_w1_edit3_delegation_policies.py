"""
Smoke tests for W1-Edit3: Agent Delegation Policies
POST /api/v1/agents/{id}/policies
"""
import pytest
import requests

# W1.5 follow-up: backend now wires POST/GET /api/v1/agents/{id}/policies.
# The persistence is JSON-on-agent.metadata; smoke validates the round-trip.

BASE = "http://localhost:8080/api/v1"


@pytest.fixture(scope="module")
def auth_headers(admin_key):
    return {"Authorization": f"Bearer {admin_key}"}


@pytest.fixture(scope="module")
def test_agent(auth_headers):
    """Create a transient agent for policy tests and delete it after."""
    payload = {
        "name": "delegation-test-agent",
        "grant_types": ["client_credentials"],
        "scopes": ["read:data"],
    }
    r = requests.post(f"{BASE}/agents", json=payload, headers=auth_headers)
    assert r.status_code in (200, 201), f"agent create failed: {r.text}"
    agent = r.json()
    yield agent
    # Cleanup
    requests.delete(f"{BASE}/agents/{agent['id']}", headers=auth_headers)


@pytest.fixture(scope="module")
def delegate_agent(auth_headers):
    """Second agent to use as delegation target."""
    payload = {
        "name": "delegation-target-agent",
        "grant_types": ["client_credentials"],
        "scopes": ["read:data"],
    }
    r = requests.post(f"{BASE}/agents", json=payload, headers=auth_headers)
    assert r.status_code in (200, 201), f"delegate agent create failed: {r.text}"
    agent = r.json()
    yield agent
    requests.delete(f"{BASE}/agents/{agent['id']}", headers=auth_headers)


# ---------------------------------------------------------------------------
# Happy path
# ---------------------------------------------------------------------------

def test_post_policies_happy_path(auth_headers, test_agent, delegate_agent):
    """POST /api/v1/agents/{id}/policies with a valid policy returns 2xx."""
    agent_id = test_agent["id"]
    delegate_id = delegate_agent["id"]

    payload = {
        "policies": [
            {"delegate_to_id": delegate_id, "scope": "read:data"},
        ]
    }
    r = requests.post(
        f"{BASE}/agents/{agent_id}/policies",
        json=payload,
        headers=auth_headers,
    )
    assert r.status_code in (200, 201, 204), (
        f"Expected 2xx, got {r.status_code}: {r.text}"
    )


def test_get_policies_after_save(auth_headers, test_agent, delegate_agent):
    """GET /api/v1/agents/{id}/policies reflects saved policies."""
    agent_id = test_agent["id"]
    delegate_id = delegate_agent["id"]

    # Ensure a policy exists
    requests.post(
        f"{BASE}/agents/{agent_id}/policies",
        json={"policies": [{"delegate_to_id": delegate_id, "scope": "read:data"}]},
        headers=auth_headers,
    )

    r = requests.get(f"{BASE}/agents/{agent_id}/policies", headers=auth_headers)
    assert r.status_code == 200, f"GET policies failed: {r.text}"
    body = r.json()
    items = body.get("data") or body.get("policies") or []
    ids = [p.get("delegate_to_id") or p.get("agent_id") for p in items]
    assert delegate_id in ids, f"Saved delegate_id not found in response: {body}"


def test_post_policies_empty_clears_all(auth_headers, test_agent):
    """POST with empty policies list removes all delegations (idempotent clear)."""
    agent_id = test_agent["id"]
    r = requests.post(
        f"{BASE}/agents/{agent_id}/policies",
        json={"policies": []},
        headers=auth_headers,
    )
    assert r.status_code in (200, 201, 204), (
        f"Expected 2xx for empty clear, got {r.status_code}: {r.text}"
    )


# ---------------------------------------------------------------------------
# Negative paths
# ---------------------------------------------------------------------------

def test_post_policies_requires_auth(test_agent):
    """POST /api/v1/agents/{id}/policies without auth returns 401."""
    agent_id = test_agent["id"]
    r = requests.post(
        f"{BASE}/agents/{agent_id}/policies",
        json={"policies": []},
    )
    assert r.status_code == 401, f"Expected 401 without auth, got {r.status_code}"


def test_post_policies_malformed_body(auth_headers, test_agent):
    """POST /api/v1/agents/{id}/policies with malformed body returns 4xx."""
    agent_id = test_agent["id"]
    r = requests.post(
        f"{BASE}/agents/{agent_id}/policies",
        json={"bad_field": "totally wrong"},
        headers=auth_headers,
    )
    assert 400 <= r.status_code < 500, (
        f"Expected 4xx for malformed body, got {r.status_code}: {r.text}"
    )


def test_get_unknown_agent_returns_404(auth_headers):
    """GET /api/v1/agents/{id}/policies on unknown agent returns 404."""
    r = requests.get(
        f"{BASE}/agents/00000000-0000-0000-0000-000000000000/policies",
        headers=auth_headers,
    )
    assert r.status_code == 404, (
        f"Expected 404 for unknown agent, got {r.status_code}: {r.text}"
    )


def test_post_unknown_agent_returns_404(auth_headers):
    """POST /api/v1/agents/{id}/policies on unknown agent returns 404."""
    r = requests.post(
        f"{BASE}/agents/00000000-0000-0000-0000-000000000000/policies",
        json={"policies": []},
        headers=auth_headers,
    )
    assert r.status_code == 404, (
        f"Expected 404 for unknown agent, got {r.status_code}: {r.text}"
    )
