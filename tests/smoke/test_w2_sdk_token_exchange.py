"""Smoke tests for OAuthClient.token_exchange() — RFC 8693.

Requires a live shark instance. The orchestrator runs these via pytest;
do NOT run locally from worktrees.

Cases:
1. Happy path: get parent token → exchange to narrower scope → verify new
   token against /api/v1/auth/me, cnf.jkt matches prover.thumbprint().
2. Negative: revoked subject_token → 401 OAuthError.
"""

from __future__ import annotations

import pytest

pytestmark = pytest.mark.skip(reason="W+1: SDK token_exchange smoke depends on agent-create method not in SDK + env vars not wired by conftest.")

from shark_auth.dpop import DPoPProver
from shark_auth.errors import OAuthError
from shark_auth.oauth import OAuthClient, Token


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(scope="module")
def prover() -> DPoPProver:
    """Single EC keypair reused across the module."""
    return DPoPProver.generate()


@pytest.fixture(scope="module")
def oauth_client(shark_base_url: str) -> OAuthClient:
    """OAuthClient pointed at the live shark fixture."""
    return OAuthClient(base_url=shark_base_url)


@pytest.fixture(scope="module")
def parent_token(
    oauth_client: OAuthClient,
    prover: DPoPProver,
    registered_agent: dict,
) -> Token:
    """Parent token with full scope obtained via client_credentials."""
    return oauth_client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id=registered_agent["client_id"],
        client_secret=registered_agent["client_secret"],
        scope="mcp:read mcp:write",
    )


# ---------------------------------------------------------------------------
# 1. Happy path
# ---------------------------------------------------------------------------

def test_token_exchange_happy_path(
    oauth_client: OAuthClient,
    prover: DPoPProver,
    parent_token: Token,
    shark_base_url: str,
) -> None:
    """Exchange parent token to narrower scope; validate via /api/v1/auth/me."""
    import requests

    child = oauth_client.token_exchange(
        subject_token=parent_token.access_token,
        dpop_prover=prover,
        scope="mcp:read",
        audience=shark_base_url,
    )

    assert isinstance(child, Token)
    assert child.access_token
    assert child.scope == "mcp:read"

    # cnf.jkt must be unchanged — same keypair bound.
    assert child.cnf_jkt == prover.thumbprint()

    # New token authenticates successfully against /api/v1/auth/me
    # with a fresh DPoP proof (same prover, new nonce/ath).
    me_url = f"{shark_base_url}/api/v1/auth/me"
    dpop_proof = prover.make_proof(htm="GET", htu=me_url, ath=child.access_token)

    resp = requests.get(
        me_url,
        headers={
            "Authorization": f"DPoP {child.access_token}",
            "DPoP": dpop_proof,
        },
    )
    assert resp.status_code == 200, f"/api/v1/auth/me returned {resp.status_code}: {resp.text}"


# ---------------------------------------------------------------------------
# 2. Negative: revoked subject_token → 401 OAuthError
# ---------------------------------------------------------------------------

def test_token_exchange_revoked_subject(
    oauth_client: OAuthClient,
    prover: DPoPProver,
    parent_token: Token,
) -> None:
    """Exchange with a revoked token raises OAuthError(invalid_token, 401)."""
    # First revoke the parent token.
    oauth_client.revoke_token(parent_token.access_token, token_type_hint="access_token")

    with pytest.raises(OAuthError) as exc_info:
        oauth_client.token_exchange(
            subject_token=parent_token.access_token,
            dpop_prover=prover,
            scope="mcp:read",
        )

    err = exc_info.value
    assert err.status_code == 401
    assert err.error == "invalid_token"
