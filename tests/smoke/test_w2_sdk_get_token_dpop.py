"""Smoke tests for OAuthClient.get_token_with_dpop() — W2 Method 1.

DO NOT RUN via pytest locally — these tests hit a live SharkAuth server and
require a running shark instance.  Run via the nightly smoke harness only.

Environment variables expected:
  SHARK_BASE_URL    — e.g. https://auth.example.com
  SHARK_ADMIN_KEY   — sk_live_... admin API key
"""

from __future__ import annotations

import os

import pytest

pytestmark = pytest.mark.skip(reason="W+1: SDK lacks agent-create method; tests rely on env vars not wired by conftest. Functional via admin HTTP in test_cascade_revoke.py")

from shark_auth.dpop import DPoPProver
from shark_auth.errors import OAuthError
from shark_auth.oauth import OAuthClient, Token

BASE_URL = os.environ.get("SHARK_BASE_URL", "http://localhost:9090")
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")


@pytest.fixture(scope="module")
def prover() -> DPoPProver:
    return DPoPProver.generate()


@pytest.fixture(scope="module")
def oauth_client() -> OAuthClient:
    return OAuthClient(base_url=BASE_URL, token=ADMIN_KEY)


@pytest.fixture(scope="module")
def registered_agent(oauth_client: OAuthClient) -> dict:
    """Register a short-lived test agent via the admin API and return its creds."""
    import requests

    resp = requests.post(
        f"{BASE_URL}/admin/agents",
        headers={"Authorization": f"Bearer {ADMIN_KEY}"},
        json={
            "name": "smoke-w2-dpop-agent",
            "grant_types": ["client_credentials"],
            "scope": "read",
        },
        timeout=10,
    )
    resp.raise_for_status()
    return resp.json()  # {"client_id": "...", "client_secret": "..."}


# ---------------------------------------------------------------------------
# Happy path
# ---------------------------------------------------------------------------

def test_get_token_happy_path(
    oauth_client: OAuthClient,
    prover: DPoPProver,
    registered_agent: dict,
) -> None:
    """client_credentials grant returns a Token with access_token + cnf_jkt."""
    token = oauth_client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id=registered_agent["client_id"],
        client_secret=registered_agent["client_secret"],
        scope="read",
    )

    assert isinstance(token, Token)
    assert token.access_token, "access_token must be non-empty"
    assert token.token_type.lower() == "dpop"
    assert token.cnf_jkt == prover.jkt, (
        f"cnf_jkt {token.cnf_jkt!r} does not match prover thumbprint {prover.jkt!r}"
    )


# ---------------------------------------------------------------------------
# Negative path
# ---------------------------------------------------------------------------

def test_invalid_client_secret_raises_oauth_error(
    oauth_client: OAuthClient,
    prover: DPoPProver,
    registered_agent: dict,
) -> None:
    """Wrong client_secret must raise OAuthError(error='invalid_client')."""
    with pytest.raises(OAuthError) as exc_info:
        oauth_client.get_token_with_dpop(
            grant_type="client_credentials",
            dpop_prover=prover,
            client_id=registered_agent["client_id"],
            client_secret="definitely-wrong-secret",
            scope="read",
        )

    err = exc_info.value
    assert err.error == "invalid_client", (
        f"Expected error='invalid_client', got {err.error!r}"
    )
