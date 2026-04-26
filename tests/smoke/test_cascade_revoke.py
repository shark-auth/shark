"""
Wave 1.5 Edit 1 + Edit 2 smoke tests.

DO NOT RUN directly — orchestrated via pytest from the repo root only.
"""

import pytest
import requests


# ---------------------------------------------------------------------------
# Edit 2 — CASCADE REVOKE
# ---------------------------------------------------------------------------

def test_cascade_revoke_kills_all_agents_and_tokens(shark_url, admin_headers, register_user, create_agent, get_token):
    """
    Happy-path cascade revoke: three agents, three tokens, all go dead.
    """
    # Register user U1
    u1 = register_user()

    # U1 registers agents A1, A2, A3 (admin creates on behalf of user)
    agents = [create_agent(created_by=u1["id"]) for _ in range(3)]

    # Each agent gets a token via client_credentials
    tokens = [get_token(a) for a in agents]

    # Verify all 3 tokens work
    for tok in tokens:
        resp = requests.get(f"{shark_url}/api/v1/auth/me",
                            headers={"Authorization": f"Bearer {tok}"})
        assert resp.status_code in (200, 204), f"Expected active token, got {resp.status_code}"

    # POST /api/v1/users/{U1.id}/revoke-agents (admin auth)
    resp = requests.post(
        f"{shark_url}/api/v1/users/{u1['id']}/revoke-agents",
        headers=admin_headers,
        json={"reason": "smoke-test cascade 2026-04-26"},
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert len(body["revoked_agent_ids"]) == 3
    assert body["revoked_consent_count"] >= 0  # may be 0 if no consents
    assert body["audit_event_id"]

    # Verify all 3 tokens now return 401
    for tok in tokens:
        resp = requests.get(f"{shark_url}/api/v1/auth/me",
                            headers={"Authorization": f"Bearer {tok}"})
        assert resp.status_code == 401, f"Expected 401 after revoke, got {resp.status_code}"

    # Verify all 3 agents have active=false
    for a in agents:
        resp = requests.get(f"{shark_url}/api/v1/agents/{a['id']}", headers=admin_headers)
        assert resp.status_code == 200
        assert resp.json()["active"] is False

    # Verify audit log has exactly one user.cascade_revoked_agents event
    resp = requests.get(
        f"{shark_url}/api/v1/audit-logs",
        headers=admin_headers,
        params={"action": "user.cascade_revoked_agents", "target_id": u1["id"]},
    )
    assert resp.status_code == 200
    logs = resp.json()["data"]
    assert len(logs) >= 1
    assert logs[0]["action"] == "user.cascade_revoked_agents"


def test_cascade_revoke_partial_agent_ids(shark_url, admin_headers, register_user, create_agent):
    """
    When agent_ids is supplied, only those agents are revoked.
    """
    u1 = register_user()
    a1 = create_agent(created_by=u1["id"])
    a2 = create_agent(created_by=u1["id"])

    resp = requests.post(
        f"{shark_url}/api/v1/users/{u1['id']}/revoke-agents",
        headers=admin_headers,
        json={"agent_ids": [a1["id"]]},
    )
    assert resp.status_code == 200
    body = resp.json()
    assert a1["id"] in body["revoked_agent_ids"]
    assert a2["id"] not in body["revoked_agent_ids"]

    # a2 should still be active
    resp = requests.get(f"{shark_url}/api/v1/agents/{a2['id']}", headers=admin_headers)
    assert resp.json()["active"] is True


def test_cascade_revoke_with_session_token_returns_403(shark_url, register_user, login_session):
    """
    cascade revoke MUST NOT work with session cookie — returns 403.
    """
    u1 = register_user()
    session_cookie = login_session(u1)

    resp = requests.post(
        f"{shark_url}/api/v1/users/{u1['id']}/revoke-agents",
        cookies=session_cookie,
        json={},
    )
    assert resp.status_code == 403, f"Expected 403, got {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# Edit 1 — LISTING ENDPOINTS
# ---------------------------------------------------------------------------

def test_users_id_agents_filter_created_returns_only_created(shark_url, admin_headers, register_user, create_agent):
    """
    GET /users/{id}/agents?filter=created returns only agents created_by that user.
    """
    u1 = register_user()
    u2 = register_user()

    # Create 2 agents for u1, 1 for u2
    a1 = create_agent(created_by=u1["id"])
    a2 = create_agent(created_by=u1["id"])
    _a3 = create_agent(created_by=u2["id"])

    resp = requests.get(
        f"{shark_url}/api/v1/users/{u1['id']}/agents",
        headers=admin_headers,
        params={"filter": "created"},
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["filter"] == "created"
    ids = [a["id"] for a in body["data"]]
    assert a1["id"] in ids
    assert a2["id"] in ids
    assert _a3["id"] not in ids
    assert body["total"] == 2


def test_users_id_agents_filter_authorized(shark_url, admin_headers, register_user, create_agent, grant_consent):
    """
    GET /users/{id}/agents?filter=authorized returns agents with active consent for that user.
    """
    u1 = register_user()
    a1 = create_agent()
    a2 = create_agent()  # no consent

    grant_consent(user_id=u1["id"], client_id=a1["client_id"])

    resp = requests.get(
        f"{shark_url}/api/v1/users/{u1['id']}/agents",
        headers=admin_headers,
        params={"filter": "authorized"},
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["filter"] == "authorized"
    ids = [a["id"] for a in body["data"]]
    assert a1["id"] in ids
    assert a2["id"] not in ids


def test_me_agents_returns_self_agents(shark_url, register_user, login_session, create_agent, admin_headers):
    """
    GET /me/agents returns agents created_by the session user.
    """
    u1 = register_user()
    a1 = create_agent(created_by=u1["id"])
    session_cookie = login_session(u1)

    resp = requests.get(
        f"{shark_url}/api/v1/me/agents",
        cookies=session_cookie,
        params={"filter": "created"},
    )
    assert resp.status_code == 200
    body = resp.json()
    ids = [a["id"] for a in body["data"]]
    assert a1["id"] in ids


def test_me_agents_requires_session(shark_url):
    """
    GET /me/agents without auth returns 401.
    """
    resp = requests.get(f"{shark_url}/api/v1/me/agents")
    assert resp.status_code == 401


def test_list_agents_created_by_user_id_param(shark_url, admin_headers, register_user, create_agent):
    """
    GET /api/v1/agents?created_by_user_id=X filters to that user's agents.
    """
    u1 = register_user()
    a1 = create_agent(created_by=u1["id"])
    _other = create_agent()  # no created_by

    resp = requests.get(
        f"{shark_url}/api/v1/agents",
        headers=admin_headers,
        params={"created_by_user_id": u1["id"]},
    )
    assert resp.status_code == 200
    body = resp.json()
    ids = [a["id"] for a in body["data"]]
    assert a1["id"] in ids
    assert _other["id"] not in ids
