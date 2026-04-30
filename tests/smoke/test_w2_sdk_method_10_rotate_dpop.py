"""Smoke test — W2 Method 10: AgentsClient.rotate_dpop_key()

Flow:
1. Register agent with initial DPoP key.
2. Issue a DPoP-bound token — assert it works.
3. Generate a new keypair.
4. rotate_dpop_key → assert result fields.
5. Introspect old token → must be inactive (revoked).
"""

from __future__ import annotations

import pytest
import requests

from shark_auth import (
    AgentsClient,
    DPoPProver,
    DPoPRotationResult,
    OAuthClient,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _issue_dpop_token(
    base_url: str,
    client_id: str,
    client_secret: str,
    prover: DPoPProver,
    scope: str = "read",
) -> str:
    """Issue a DPoP client_credentials token. Returns access_token string."""
    token_endpoint = f"{base_url}/oauth/token"
    proof = prover.make_proof(htm="POST", htu=token_endpoint)
    resp = requests.post(
        token_endpoint,
        data={
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
            "scope": scope,
        },
        headers={"DPoP": proof},
        timeout=10,
    )
    assert resp.status_code == 200, f"token issue failed ({resp.status_code}): {resp.text}"
    return resp.json()["access_token"]


# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------


def test_rotate_dpop_key(base_url: str, admin_key: str) -> None:
    """Full rotation lifecycle: issue → rotate → verify old token revoked."""
    agents = AgentsClient(base_url=base_url, token=admin_key)
    oauth = OAuthClient(base_url=base_url, token=admin_key)

    # 1. Register agent.
    prover_old = DPoPProver.generate()
    agent = agents.register_agent(
        app_id="smoke_w2m10",
        name="smoke_rotate_dpop_agent",
        scopes=["read"],
        token_endpoint_auth_method="client_secret_post",
        # Embed the initial DPoP public key in metadata so the backend
        # can derive old_jkt on rotation.
        metadata={"dpop_public_jwk": prover_old.public_jwk},
    )
    client_id = agent["client_id"]
    client_secret = agent.get("client_secret", "")
    agent_id = agent["id"]

    try:
        # 2. Issue a DPoP-bound token using the initial key.
        old_token = _issue_dpop_token(base_url, client_id, client_secret, prover_old)
        assert old_token, "initial DPoP token must be non-empty"

        # Verify it is currently active.
        info = oauth.introspect_token(old_token)
        assert info.get("active") is True, f"initial token should be active: {info}"

        # 3. Generate a new keypair.
        prover_new = DPoPProver.generate()
        new_pub_jwk = prover_new.public_jwk

        # 4. Rotate the DPoP key.
        result = agents.rotate_dpop_key(
            agent_id,
            new_public_key_jwk=new_pub_jwk,
            reason="smoke test W2M10 2026-04-26",
        )

        # Assert result shape.
        assert isinstance(result, DPoPRotationResult), (
            f"Expected DPoPRotationResult, got {type(result)}"
        )
        assert result.new_jkt, "new_jkt must be non-empty"
        assert result.audit_event_id, "audit_event_id must be non-empty"
        assert isinstance(result.revoked_token_count, int)
        # old_jkt will be populated because we embedded it in metadata at registration.
        # Accept both populated and empty (server may not have a prior JWK).
        assert isinstance(result.old_jkt, str)

        # old_jkt and new_jkt must differ (rotation produced a new binding).
        if result.old_jkt:
            assert result.old_jkt != result.new_jkt, (
                "old_jkt and new_jkt must differ after rotation"
            )

        # 5. Old token must now be inactive (rotation revokes all bound tokens).
        info_after = oauth.introspect_token(old_token)
        assert info_after.get("active") is False, (
            f"old token should be revoked after rotation, got: {info_after}"
        )

        # (Bonus) Issue a new token with the rotated key — should succeed.
        # The server binds future tokens to the new jkt stored in Metadata.
        new_token = _issue_dpop_token(
            base_url, client_id, client_secret, prover_new
        )
        assert new_token, "new DPoP token with rotated key must be non-empty"

    finally:
        try:
            agents.revoke_agent(agent_id)
        except Exception:
            pass
