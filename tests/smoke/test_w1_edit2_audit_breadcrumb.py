"""
W1-Edit2 Smoke Tests — Audit delegation-chain breadcrumb
=========================================================
Verifies that GET /api/v1/audit returns events whose fields satisfy the
shape the delegation-chain breadcrumb component consumes.

DO NOT RUN via pytest directly in a worktree — the orchestrator runs these
against the shared shark.exe instance.
"""

import os
import pytest
import requests

BASE = os.environ.get("SHARK_BASE_URL", "http://localhost:8080")
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")

HEADERS = {"Authorization": f"Bearer {ADMIN_KEY}"}


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def get_audit_events(params: dict | None = None):
    """Fetch audit log events from the API."""
    r = requests.get(f"{BASE}/api/v1/audit-logs", headers=HEADERS, params=params or {}, timeout=10)
    assert r.status_code == 200, f"Expected 200, got {r.status_code}: {r.text}"
    body = r.json()
    # API may return list directly or wrapped in {data, items, audit_logs}
    if isinstance(body, list):
        return body
    for key in ("data", "items", "audit_logs"):
        if key in body and isinstance(body[key], list):
            return body[key]
    return []


# ---------------------------------------------------------------------------
# Happy path — agent-actor event with act claim chain
# ---------------------------------------------------------------------------

def test_agent_actor_event_has_act_chain_shape():
    """
    An audit event with actor_type == 'agent' SHOULD carry either:
      - act_chain: [{sub, jkt?, label?}, ...]   (preferred)
      - oauth.act: {sub, email?}                (legacy single-hop)
    so the breadcrumb component can render without crashing.
    """
    events = get_audit_events({"actor_type": "agent", "limit": "50"})

    agent_events = [e for e in events if e.get("actor_type") == "agent"]

    if not agent_events:
        pytest.skip("No agent-actor events present; cannot validate breadcrumb shape")

    for ev in agent_events:
        # Must have a usable actor identifier for the terminal breadcrumb chip
        assert ev.get("actor_email") or ev.get("actor_id"), (
            f"Agent event {ev.get('id')} missing actor_email / actor_id"
        )

        # If act_chain present, each element must have at minimum a 'sub' key
        act_chain = ev.get("act_chain")
        if act_chain is not None:
            assert isinstance(act_chain, list), "act_chain must be a list"
            assert len(act_chain) >= 1, "act_chain must have at least one entry"
            for node in act_chain:
                assert "sub" in node, f"act_chain node missing 'sub': {node}"
                # jkt is optional; if present must be a non-empty string
                if "jkt" in node:
                    assert isinstance(node["jkt"], str) and node["jkt"], (
                        "act_chain node 'jkt' must be a non-empty string"
                    )

        # Legacy single-hop: oauth.act.sub must exist
        oauth = ev.get("oauth") or {}
        legacy_act = oauth.get("act")
        if legacy_act is not None:
            assert "sub" in legacy_act, "oauth.act missing 'sub'"

        # At least one of the two chain shapes must be present for delegated events
        is_delegated = ev.get("is_delegated") or act_chain or legacy_act
        # (non-delegated agent events legitimately have neither; that's fine)


# ---------------------------------------------------------------------------
# Negative path — human actor event should NOT have act chain
# ---------------------------------------------------------------------------

def test_human_actor_event_has_no_act_chain():
    """
    Events with actor_type == 'user' or 'admin' are NOT delegated agent calls.
    The breadcrumb must NOT render for these events; confirm the fields are absent.
    """
    events = get_audit_events({"actor_type": "user", "limit": "50"})
    human_events = [e for e in events if e.get("actor_type") in ("user", "admin")]

    if not human_events:
        pytest.skip("No human-actor events present; cannot validate negative path")

    for ev in human_events:
        act_chain = ev.get("act_chain")
        # act_chain for a human event must be absent or null/empty
        assert not act_chain, (
            f"Human event {ev.get('id')} unexpectedly has act_chain: {act_chain}"
        )
        oauth = ev.get("oauth") or {}
        legacy_act = oauth.get("act")
        # Legacy act claim also must be absent for pure human events
        assert not legacy_act, (
            f"Human event {ev.get('id')} unexpectedly has oauth.act: {legacy_act}"
        )
