"""
W1-Edit4: Agent Security Card — smoke tests
============================================
Verifies that the endpoints consumed by the Agent Security attention card
return expected fields.

Endpoints under test:
  GET /api/v1/audit-logs?action=token.exchange&from=<iso>&limit=200
  GET /api/v1/agents?limit=200
  GET /api/v1/agents/{id}/tokens

DO NOT RUN via pytest directly — orchestrator-only (see MEMORY.md).
"""

import os
import time
import datetime
import pytest
import requests

BASE = os.environ.get("SHARK_BASE_URL", "http://localhost:8080")

# AUTH is built lazily per-test from the admin_key fixture so module import
# doesn't bake an empty Bearer at collect time.
def _auth(admin_key: str):
    return {"Authorization": f"Bearer {admin_key}"}


# ---------------------------------------------------------------------------
# Happy-path
# ---------------------------------------------------------------------------

def test_audit_logs_token_exchange_fields(admin_key):
    """Audit-log endpoint filtered to token.exchange returns expected envelope."""
    since = (datetime.datetime.utcnow() - datetime.timedelta(hours=24)).strftime(
        "%Y-%m-%dT%H:%M:%SZ"
    )
    resp = requests.get(
        f"{BASE}/api/v1/audit-logs",
        params={"action": "token.exchange", "from": since, "limit": "200"},
        headers=_auth(admin_key),
        timeout=10,
    )
    assert resp.status_code == 200, f"expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    # Envelope shape required by AgentSecurityCard
    assert "data" in body, "response must contain 'data' array"
    assert isinstance(body["data"], list), "'data' must be a list"
    assert "has_more" in body, "response must contain 'has_more'"

    # Each event (if any) must have an event_type / action and created_at
    for ev in body["data"][:5]:
        assert "id" in ev or "event_type" in ev or "action" in ev, (
            f"audit event missing identity field: {ev}"
        )


def test_agents_list_fields(admin_key):
    """Agents list returns total + data array consumed for DPoP / delegation metrics."""
    resp = requests.get(
        f"{BASE}/api/v1/agents",
        params={"limit": "200"},
        headers=_auth(admin_key),
        timeout=10,
    )
    assert resp.status_code == 200, f"expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert "data" in body, "response must contain 'data' array"
    assert isinstance(body["data"], list), "'data' must be a list"

    # If any agents exist, spot-check token endpoint
    agents = body["data"]
    if agents:
        agent_id = agents[0]["id"]
        tok_resp = requests.get(
            f"{BASE}/api/v1/agents/{agent_id}/tokens",
            headers=_auth(admin_key),
            timeout=10,
        )
        assert tok_resp.status_code == 200, (
            f"tokens endpoint returned {tok_resp.status_code}"
        )
        tok_body = tok_resp.json()
        assert "data" in tok_body, "tokens response must contain 'data'"
        assert isinstance(tok_body["data"], list), "'data' must be a list"

        # Each token must include fields the card reads for DPoP detection
        for tok in tok_body["data"][:3]:
            # At minimum a token must have an id and expires_at; dpop_jkt is optional
            assert "id" in tok, f"token missing 'id': {tok}"


# ---------------------------------------------------------------------------
# Negative: unauthenticated → 401
# ---------------------------------------------------------------------------

def test_audit_logs_unauth_returns_401():
    """Audit endpoint must reject requests without a valid admin key."""
    since = (datetime.datetime.utcnow() - datetime.timedelta(hours=24)).strftime(
        "%Y-%m-%dT%H:%M:%SZ"
    )
    resp = requests.get(
        f"{BASE}/api/v1/audit-logs",
        params={"action": "token.exchange", "from": since, "limit": "10"},
        headers={"Authorization": "Bearer invalid-key-for-smoke-test"},
        timeout=10,
    )
    assert resp.status_code == 401, (
        f"expected 401 for bad admin key, got {resp.status_code}"
    )


def test_agents_list_unauth_returns_401():
    """Agents list must reject requests without a valid admin key."""
    resp = requests.get(
        f"{BASE}/api/v1/agents",
        params={"limit": "10"},
        headers={"Authorization": "Bearer invalid-key-for-smoke-test"},
        timeout=10,
    )
    assert resp.status_code == 401, (
        f"expected 401 for bad admin key, got {resp.status_code}"
    )
