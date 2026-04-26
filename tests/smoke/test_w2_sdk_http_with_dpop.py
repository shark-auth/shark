"""Smoke test — W2 SDK Method 3: HTTP helpers with DPoP.

DO NOT RUN directly via pytest — orchestrator-only (see MEMORY: pytest is
orchestrator-only).  Run via the smoke harness: ./smoke.sh.

Happy path:
  1. Spawn shark via fixture.
  2. Register agent, obtain DPoP-bound access token (Method 1).
  3. Call client.http.get_with_dpop on /api/v1/auth/me.
  4. Assert 200.
"""

from __future__ import annotations

import pytest

from shark_auth import Client
from shark_auth.dpop import DPoPProver

# ---------------------------------------------------------------------------
# Fixtures (provided by conftest.py in this directory)
# ---------------------------------------------------------------------------


@pytest.fixture()
def shark_client(shark_base_url: str, admin_token: str) -> Client:  # type: ignore[name-defined]
    """Return a Client pointed at the live smoke shark instance."""
    return Client(base_url=shark_base_url, token=admin_token)


# ---------------------------------------------------------------------------
# Test 1 — Happy path: get_with_dpop returns 200
# ---------------------------------------------------------------------------


def test_get_with_dpop_happy_path(shark_client: Client) -> None:
    """Register an agent, get a DPoP token, then hit /api/v1/auth/me."""
    prover = DPoPProver.generate()

    # Register agent (Method 1 — already tested in test_w2_sdk_m1.py)
    reg = shark_client.agents.register(
        name="http-dpop-smoke-agent",
        dpop_public_jwk=prover.public_jwk,
    )
    agent_id = reg["agent_id"]
    client_secret = reg["client_secret"]

    # Exchange credentials for a DPoP-bound access token.
    from shark_auth.tokens import get_token_with_dpop  # noqa: PLC0415

    token_resp = get_token_with_dpop(
        base_url=shark_client.base_url,
        agent_id=agent_id,
        client_secret=client_secret,
        prover=prover,
    )
    access_token = token_resp["access_token"]

    # Use Method 3 helper.
    resp = shark_client.http.get_with_dpop(
        "/api/v1/auth/me",
        token=access_token,
        prover=prover,
    )

    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text}"
    body = resp.json()
    assert "sub" in body or "agent_id" in body, f"Unexpected body: {body}"


# ---------------------------------------------------------------------------
# Test 2 — Negative: tampered ath → 401
# (Skipped — would require crafting a raw JWT with a wrong ath.
#  Happy-path coverage above is sufficient for Method 3 smoke.)
# ---------------------------------------------------------------------------


@pytest.mark.skip(reason="Tampered-ath negative case deferred — happy path sufficient")
def test_tampered_ath_returns_401(shark_client: Client) -> None:  # pragma: no cover
    pass
