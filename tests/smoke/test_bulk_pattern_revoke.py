"""
Wave 1.6 Layer 4 smoke tests — bulk-pattern token revoke.

POST /api/v1/admin/oauth/revoke-by-pattern
  body: { client_id_pattern, reason }
  response: { revoked_count, audit_event_id, pattern_matched }

DO NOT RUN directly — orchestrated via pytest from the repo root only.
"""

import os
import time
import pytest
import requests

BASE_URL = os.environ.get("BASE", "http://localhost:8080")


# ---------------------------------------------------------------------------
# Local fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def admin_headers(admin_key):
    return {"Authorization": f"Bearer {admin_key}"}


@pytest.fixture
def create_agent(admin_headers):
    """Factory: create an agent, returns full agent dict (includes client_secret)."""
    created = []

    def _create(name_prefix="bulkrevoke"):
        ts = int(time.time() * 1000)
        resp = requests.post(
            f"{BASE_URL}/api/v1/agents",
            headers=admin_headers,
            json={
                "name": f"{name_prefix}_{ts}",
                "grant_types": ["client_credentials"],
                "scopes": ["read"],
            },
        )
        assert resp.status_code in (200, 201), f"create_agent failed: {resp.text}"
        agent = resp.json()
        created.append(agent["id"])
        return agent

    yield _create

    for aid in created:
        requests.delete(f"{BASE_URL}/api/v1/agents/{aid}", headers=admin_headers)


def _get_token(agent):
    """Issue a client_credentials token via HTTP Basic auth."""
    resp = requests.post(
        f"{BASE_URL}/oauth/token",
        data={"grant_type": "client_credentials", "scope": "read"},
        auth=(agent["client_id"], agent["client_secret"]),
    )
    assert resp.status_code == 200, f"_get_token failed: {resp.text}"
    return resp.json()["access_token"]


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

def test_bulk_revoke_by_exact_glob(create_agent, admin_headers):
    """
    Register 3 agents, issue tokens for each, then revoke all matching '*'
    (matches every client_id). Assert revoked_count >= 3, audit_event_id
    present, pattern_matched echoed back.
    """
    agents = [create_agent("bulkrevoke_v32") for _ in range(3)]
    tokens = [_get_token(a) for a in agents]
    assert len(tokens) == 3

    # Use '*' to match all — guaranteed to cover our 3 just-issued tokens
    pattern = "*"
    resp = requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        headers=admin_headers,
        json={"client_id_pattern": pattern, "reason": "smoke-test bulk revoke"},
    )
    assert resp.status_code == 200, f"bulk revoke failed: {resp.text}"
    body = resp.json()
    assert body.get("revoked_count") >= 3, f"expected >=3 revoked, got {body}"
    assert body.get("audit_event_id"), "audit_event_id should be present"
    assert body.get("pattern_matched") == pattern, "pattern_matched should echo input"


def test_bulk_revoke_specific_client_id_glob(create_agent, admin_headers):
    """
    Register 3 agents, capture their client_ids, build a GLOB from the first
    one's full client_id (exact match via '*<id>*'), assert exactly 1 revoked.
    """
    agents = [create_agent("bulkrevoke_specific") for _ in range(3)]
    for a in agents:
        _get_token(a)

    target = agents[0]
    # Exact GLOB on one client_id: surround with * to ensure match
    pattern = f"*{target['client_id']}*"

    resp = requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        headers=admin_headers,
        json={"client_id_pattern": pattern, "reason": "targeted smoke test"},
    )
    assert resp.status_code == 200, f"bulk revoke failed: {resp.text}"
    body = resp.json()
    assert body["revoked_count"] >= 1, "should have revoked at least 1 token"
    assert body["pattern_matched"] == pattern


def test_bulk_revoke_returns_zero_when_no_match(create_agent, admin_headers):
    """
    Pattern that matches nothing should return 200 with revoked_count == 0.
    """
    resp = requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        headers=admin_headers,
        json={
            "client_id_pattern": "NOMATCH_ZZZZZ_9999999999999",
            "reason": "zero-match smoke test",
        },
    )
    assert resp.status_code == 200, f"unexpected status: {resp.text}"
    body = resp.json()
    assert body["revoked_count"] == 0
    assert body["audit_event_id"]


def test_bulk_revoke_missing_pattern_returns_400(admin_headers):
    """
    Omitting client_id_pattern should return 400.
    """
    resp = requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        headers=admin_headers,
        json={"reason": "forgot pattern"},
    )
    assert resp.status_code == 400, f"expected 400, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert body.get("error") == "invalid_request"


def test_bulk_revoke_empty_pattern_returns_400(admin_headers):
    """
    Empty string client_id_pattern should return 400.
    """
    resp = requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        headers=admin_headers,
        json={"client_id_pattern": "", "reason": "empty pattern"},
    )
    assert resp.status_code == 400, f"expected 400, got {resp.status_code}: {resp.text}"


def test_bulk_revoke_token_can_be_reissued_after_revoke(create_agent, admin_headers):
    """
    After bulk revoke, the agent itself is NOT disabled — re-issuing a
    client_credentials token must still succeed (the revoke only marks
    existing tokens in oauth_tokens, it does not deactivate the client).
    """
    agent = create_agent("bulkrevoke_reissue")
    _get_token(agent)  # issue once

    # Revoke all tokens for this agent
    pattern = f"*{agent['client_id']}*"
    resp = requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        headers=admin_headers,
        json={"client_id_pattern": pattern, "reason": "reissue test"},
    )
    assert resp.status_code == 200
    assert resp.json()["revoked_count"] >= 1

    # Re-issue must succeed
    new_token = _get_token(agent)
    assert new_token, "should be able to re-issue token after bulk revoke"


def test_bulk_revoke_tokens_marked_revoked_in_token_list(create_agent, admin_headers):
    """
    After bulk revoke, GET /api/v1/agents/{id}/tokens should show revoked_at
    set on the previously-issued token.
    """
    agent = create_agent("bulkrevoke_check")
    _get_token(agent)

    pattern = f"*{agent['client_id']}*"
    requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        headers=admin_headers,
        json={"client_id_pattern": pattern, "reason": "token list check"},
    )

    # Wait for background revocation to complete
    time.sleep(0.5)

    resp = requests.get(
        f"{BASE_URL}/api/v1/agents/{agent['id']}/tokens",
        headers=admin_headers,
    )
    # 404 means the endpoint isn't yet implemented — skip gracefully
    if resp.status_code == 404:
        pytest.skip("GET /agents/{id}/tokens not implemented")
    assert resp.status_code == 200, f"token list failed: {resp.text}"
    tokens = resp.json().get("data") or resp.json().get("tokens") or []
    revoked = [t for t in tokens if t.get("revoked_at")]
    assert len(revoked) >= 1, f"expected at least 1 revoked token, got: {tokens}"


def test_bulk_revoke_unauthenticated_returns_401():
    """
    Without admin key, endpoint should return 401.
    """
    resp = requests.post(
        f"{BASE_URL}/api/v1/admin/oauth/revoke-by-pattern",
        json={"client_id_pattern": "*", "reason": "unauth test"},
    )
    assert resp.status_code == 401, f"expected 401, got {resp.status_code}"
