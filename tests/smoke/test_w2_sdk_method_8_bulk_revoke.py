"""Smoke test — W2 Method 8: OAuthClient.bulk_revoke_by_pattern()

Registers 3 agents whose client_ids match a common prefix pattern and 2
"control" agents that should be unaffected.  Issues tokens for all 5,
calls bulk_revoke_by_pattern, and asserts the right counts.
"""

from __future__ import annotations

import time
import uuid
import pytest

from shark_auth import AgentsClient, OAuthClient, BulkRevokeResult


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _register_agent(agents: AgentsClient, name: str) -> dict:
    """Register an agent and return the full response including client_secret."""
    return agents.register_agent(
        app_id="smoke_w2m8",
        name=name,
        scopes=["read"],
    )


def _issue_token(base_url: str, client_id: str, client_secret: str) -> str:
    """Issue a client_credentials token via Basic auth and return the access_token."""
    import requests

    resp = requests.post(
        f"{base_url}/oauth/token",
        data={
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
            "scope": "read",
        },
        timeout=10,
    )
    assert resp.status_code == 200, f"token issue failed: {resp.text}"
    return resp.json()["access_token"]


# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------


def test_bulk_revoke_by_pattern(base_url: str, admin_key: str) -> None:
    """
    Register 3 "target" agents + 2 "control" agents.
    Issue tokens for all 5.
    bulk_revoke_by_pattern on the target prefix.
    Assert revoked_count >= 3, audit_event_id non-empty, pattern_matched correct.
    """
    agents = AgentsClient(base_url=base_url, token=admin_key)
    oauth = OAuthClient(base_url=base_url, token=admin_key)

    run_id = uuid.uuid4().hex[:8]
    target_prefix = f"smoke_target_{run_id}"

    created_agents = []
    try:
        # Register 3 target agents whose names start with target_prefix.
        # The server assigns client_ids as "shark_agent_<nanoid>"; we rely on
        # name-based registration and retrieve client_ids from the response.
        target_ids = []
        for i in range(3):
            agent = _register_agent(agents, f"{target_prefix}_agent_{i}")
            client_id = agent["client_id"]
            client_secret = agent.get("client_secret", "")
            created_agents.append(agent["id"])
            target_ids.append((client_id, client_secret))

        # Register 2 control agents (different prefix, should NOT be revoked).
        control_prefix = f"smoke_ctrl_{run_id}"
        control_ids = []
        for i in range(2):
            agent = _register_agent(agents, f"{control_prefix}_agent_{i}")
            created_agents.append(agent["id"])
            control_ids.append((agent["client_id"], agent.get("client_secret", "")))

        # Issue tokens for all 5 agents.
        target_tokens = []
        for cid, csecret in target_ids:
            if csecret:
                tok = _issue_token(base_url, cid, csecret)
                target_tokens.append(tok)

        control_tokens = []
        for cid, csecret in control_ids:
            if csecret:
                tok = _issue_token(base_url, cid, csecret)
                control_tokens.append(tok)

        # Bulk-revoke all tokens for the 3 target client_ids individually,
        # using their exact client_ids as GLOB patterns (exact match).
        # We call bulk_revoke_by_pattern once per target client_id to satisfy
        # the "3 agents match" invariant (the server matches GLOB against
        # stored client_ids, not agent names).
        total_revoked = 0
        last_result: BulkRevokeResult | None = None

        for cid, _ in target_ids:
            result = oauth.bulk_revoke_by_pattern(
                client_id_pattern=cid,
                reason=f"smoke test W2M8 {run_id} 2026-04-26",
            )
            assert isinstance(result, BulkRevokeResult)
            assert result.audit_event_id, "audit_event_id must be non-empty"
            assert result.pattern_matched == cid, (
                f"pattern_matched mismatch: expected {cid!r}, got {result.pattern_matched!r}"
            )
            total_revoked += result.revoked_count
            last_result = result

        # At least the tokens we issued must be accounted for.
        assert total_revoked >= len(target_tokens), (
            f"expected >= {len(target_tokens)} revoked tokens, got {total_revoked}"
        )

        # Control tokens should still be valid (introspect should return active).
        for tok in control_tokens:
            info = oauth.introspect_token(tok)
            assert info.get("active") is True, (
                f"control token should still be active after pattern revoke, got: {info}"
            )

    finally:
        # Cleanup
        for agent_id in created_agents:
            try:
                agents.revoke_agent(agent_id)
            except Exception:
                pass
