"""
Wave 1 Edit 5 — Get Started: Agent Onboarding section smoke tests.

Covers the two CTA endpoints surfaced by the new Agent Onboarding checklist:
  - Create an agent  → GET /api/v1/agents
  - Verify audit     → GET /audit (hash route, served by admin SPA)

DO NOT RUN via pytest directly — orchestrator-only (see MEMORY).
"""
import pytest
import requests

BASE = "http://localhost:9000"


# ── Helpers ────────────────────────────────────────────────────────────────────

def admin_key():
    """Return admin key from env or skip."""
    import os
    key = os.getenv("SHARK_ADMIN_KEY", "")
    if not key:
        pytest.skip("SHARK_ADMIN_KEY not set")
    return key


# ── /api/v1/agents — create-agent CTA ─────────────────────────────────────────

class TestAgentsEndpoint:
    """GET /api/v1/agents must be reachable; the Agent Onboarding CTA links here."""

    def test_agents_list_happy(self):
        """Authenticated request returns 200 with a JSON body."""
        key = admin_key()
        r = requests.get(
            f"{BASE}/api/v1/agents",
            headers={"Authorization": f"Bearer {key}"},
            timeout=5,
        )
        assert r.status_code == 200, f"Expected 200, got {r.status_code}: {r.text[:200]}"
        body = r.json()
        # Shape check: list (possibly empty) or object with agents key
        assert isinstance(body, (list, dict)), f"Unexpected body type: {type(body)}"

    def test_agents_list_unauth(self):
        """Unauthenticated request must return 401 (not 200, not 500)."""
        r = requests.get(f"{BASE}/api/v1/agents", timeout=5)
        assert r.status_code == 401, f"Expected 401, got {r.status_code}: {r.text[:200]}"


# ── /audit — verify-audit CTA ─────────────────────────────────────────────────

class TestAuditEndpoint:
    """GET /audit (admin SPA hash route) must be reachable; verify-audit CTA links here."""

    def test_audit_page_reachable(self):
        """The admin SPA serves /audit (or the root index.html for the hash route)."""
        r = requests.get(f"{BASE}/audit", timeout=5, allow_redirects=True)
        # SPA may return 200 with HTML, or redirect to /admin/audit — both are fine.
        assert r.status_code in (200, 301, 302), (
            f"Expected 200/30x, got {r.status_code}: {r.text[:200]}"
        )
        # Must not be a raw 5xx
        assert r.status_code < 500, f"Server error on /audit: {r.status_code}"

    def test_audit_delegation_filter_unauth(self):
        """GET /api/v1/admin/audit?delegation=true without auth must return 401."""
        r = requests.get(
            f"{BASE}/api/v1/admin/audit",
            params={"delegation": "true"},
            timeout=5,
        )
        assert r.status_code == 401, (
            f"Expected 401 for unauth audit query, got {r.status_code}: {r.text[:200]}"
        )
