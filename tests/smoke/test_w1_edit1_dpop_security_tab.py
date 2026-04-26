"""W1-Edit1 smoke — DPoP Security tab data contract.

Verifies that GET /api/v1/agents/{id} returns the DPoP fields
consumed by the new Security tab in the admin dashboard:
  - dpop_jkt  (JWK thumbprint)
  - dpop_key_id / key_id
  - audit log contains key-rotation events (or at least is queryable)

Two cases:
  1. Happy path  — agent exists, response includes expected DPoP fields.
  2. Negative path — unknown agent ID returns 404.
"""
import os
import pytest
import requests

BASE_URL = os.environ.get("BASE", "http://localhost:8080")


def _admin_headers(admin_key: str) -> dict:
    return {"Authorization": f"Bearer {admin_key}"}


def _create_agent(admin_key: str, name: str = "dpop-security-tab-agent") -> dict:
    """Helper: create a minimal agent and return the response dict."""
    r = requests.post(
        f"{BASE_URL}/api/v1/agents",
        json={
            "name": name,
            "grant_types": ["client_credentials"],
            "scopes": ["openid"],
        },
        headers=_admin_headers(admin_key),
        timeout=10,
    )
    assert r.status_code in (200, 201), f"agent create failed {r.status_code}: {r.text}"
    return r.json()


# ── fixtures ──────────────────────────────────────────────────────────────────

@pytest.fixture(scope="module")
def admin_key(server):  # noqa: F811 — `server` fixture from conftest.py
    """Read the first-boot admin key written by conftest."""
    key_path = "data/admin.key.firstboot"
    if not os.path.exists(key_path):
        pytest.skip("First-boot admin key not found — server may not have started")
    return open(key_path).read().strip()


@pytest.fixture(scope="module")
def agent(admin_key):
    return _create_agent(admin_key, name="dpop-sec-tab-smoke")


# ── happy path ────────────────────────────────────────────────────────────────

class TestDPoPSecurityTabHappyPath:
    """GET /api/v1/agents/{id} must expose DPoP fields the Security tab uses."""

    def test_agent_detail_returns_200(self, agent, admin_key):
        agent_id = agent.get("id") or agent.get("client_id")
        r = requests.get(
            f"{BASE_URL}/api/v1/agents/{agent_id}",
            headers=_admin_headers(admin_key),
            timeout=10,
        )
        assert r.status_code == 200, f"Expected 200, got {r.status_code}: {r.text}"

    def test_agent_detail_has_id_field(self, agent, admin_key):
        agent_id = agent.get("id") or agent.get("client_id")
        r = requests.get(
            f"{BASE_URL}/api/v1/agents/{agent_id}",
            headers=_admin_headers(admin_key),
            timeout=10,
        )
        body = r.json()
        # The Security tab needs at minimum an agent id to key on
        assert "id" in body or "client_id" in body, (
            "Response must contain 'id' or 'client_id'"
        )

    def test_dpop_fields_present_or_null(self, agent, admin_key):
        """dpop_jkt and dpop_key_id may be null for a newly-created agent that has
        not yet performed a DPoP flow, but the keys must exist in the response
        so the Security tab can render them (showing '—' for null)."""
        agent_id = agent.get("id") or agent.get("client_id")
        r = requests.get(
            f"{BASE_URL}/api/v1/agents/{agent_id}",
            headers=_admin_headers(admin_key),
            timeout=10,
        )
        body = r.json()
        # Accept presence of either the canonical or alias field names
        has_jkt = "dpop_jkt" in body or "jkt" in body
        has_key_id = "dpop_key_id" in body or "key_id" in body
        # At least one DPoP field must be present (even if its value is null/None)
        assert has_jkt or has_key_id, (
            f"Response missing DPoP fields (dpop_jkt / dpop_key_id). "
            f"Keys present: {list(body.keys())}"
        )

    def test_audit_endpoint_reachable(self, agent, admin_key):
        """The Security tab's rotation history pulls from the audit endpoint."""
        agent_id = agent.get("id") or agent.get("client_id")
        r = requests.get(
            f"{BASE_URL}/api/v1/agents/{agent_id}/audit?limit=50",
            headers=_admin_headers(admin_key),
            timeout=10,
        )
        assert r.status_code == 200, (
            f"Audit endpoint returned {r.status_code}: {r.text}"
        )
        body = r.json()
        # Must return a list-style envelope
        assert "data" in body, f"Expected 'data' key in audit response, got: {list(body.keys())}"
        assert isinstance(body["data"], list), "audit 'data' must be a list"


# ── negative path ─────────────────────────────────────────────────────────────

class TestDPoPSecurityTabNegativePath:
    """Unknown agent ID must return 404."""

    def test_unknown_agent_returns_404(self, admin_key):
        r = requests.get(
            f"{BASE_URL}/api/v1/agents/does-not-exist-xyzzy-0000",
            headers=_admin_headers(admin_key),
            timeout=10,
        )
        assert r.status_code == 404, (
            f"Expected 404 for unknown agent, got {r.status_code}: {r.text}"
        )

    def test_unknown_agent_audit_returns_404(self, admin_key):
        r = requests.get(
            f"{BASE_URL}/api/v1/agents/does-not-exist-xyzzy-0000/audit",
            headers=_admin_headers(admin_key),
            timeout=10,
        )
        assert r.status_code in (404, 400), (
            f"Expected 404 or 400 for unknown agent audit, got {r.status_code}: {r.text}"
        )
