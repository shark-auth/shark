"""Smoke test — W2 Method 9: VaultClient.disconnect() + VaultClient.fetch_token()

Both tests skip gracefully when the vault infrastructure (provider catalog +
active connection) is not provisioned in the smoke environment.
"""

from __future__ import annotations

import pytest

from shark_auth import (
    AgentsClient,
    DPoPProver,
    OAuthClient,
    VaultClient,
    VaultDisconnectResult,
    VaultTokenResult,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(scope="module")
def base_url(shark_base_url: str) -> str:
    return shark_base_url


@pytest.fixture(scope="module")
def admin_key(shark_admin_key: str) -> str:
    return shark_admin_key


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _list_vault_connections(base_url: str, admin_key: str) -> list[dict]:
    """Return the list of vault connections visible to the admin."""
    import requests

    resp = requests.get(
        f"{base_url}/api/v1/admin/vault/connections",
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=10,
    )
    if resp.status_code != 200:
        return []
    data = resp.json()
    return data.get("data", data) if isinstance(data, dict) else data


# ---------------------------------------------------------------------------
# Test 1 — disconnect
# ---------------------------------------------------------------------------


def test_disconnect(base_url: str, admin_key: str) -> None:
    """Delete a vault connection and assert VaultDisconnectResult fields.

    Skips if no vault connection exists in the smoke environment.
    """
    connections = _list_vault_connections(base_url, admin_key)
    if not connections:
        pytest.skip("No vault connections available — skipping disconnect smoke test")

    # Use the first available connection.
    conn = connections[0]
    connection_id = conn.get("id") or conn.get("connection_id", "")
    assert connection_id, "Connection dict missing id field"

    vault = VaultClient(base_url=base_url, admin_key=admin_key)
    result = vault.disconnect(connection_id, cascade_to_agents=True)

    assert isinstance(result, VaultDisconnectResult)
    assert result.connection_id == connection_id, (
        f"connection_id mismatch: expected {connection_id!r}, got {result.connection_id!r}"
    )
    assert isinstance(result.revoked_agent_ids, list)
    assert isinstance(result.revoked_token_count, int)
    # cascade_audit_event_id may be None when no agents were cascade-revoked.
    assert result.cascade_audit_event_id is None or isinstance(
        result.cascade_audit_event_id, str
    )


# ---------------------------------------------------------------------------
# Test 2 — fetch_token
# ---------------------------------------------------------------------------


def test_fetch_token(base_url: str, admin_key: str) -> None:
    """Fetch a vault-brokered token using a DPoP-authenticated agent bearer.

    Skips if no vault provider is provisioned in the smoke environment.
    """
    import requests

    # Check that at least one vault provider exists.
    resp = requests.get(
        f"{base_url}/api/v1/vault/providers",
        headers={"Authorization": f"Bearer {admin_key}"},
        timeout=10,
    )
    if resp.status_code != 200:
        pytest.skip("Vault providers endpoint unavailable — skipping fetch_token smoke test")

    data = resp.json()
    providers = data.get("data", data) if isinstance(data, dict) else data
    if not providers:
        pytest.skip("No vault providers provisioned — skipping fetch_token smoke test")

    provider_slug = providers[0].get("slug") or providers[0].get("name", "")
    if not provider_slug:
        pytest.skip("Provider slug not available — skipping fetch_token smoke test")

    # Register a temporary agent with vault:read scope.
    agents = AgentsClient(base_url=base_url, token=admin_key)
    prover = DPoPProver.generate()
    agent = agents.register_agent(
        app_id="smoke_w2m9",
        name="smoke_vault_fetch_agent",
        scopes=["vault:read"],
    )
    client_id = agent["client_id"]
    client_secret = agent.get("client_secret", "")
    agent_id = agent["id"]

    try:
        # Issue a DPoP-bound token.
        oauth = OAuthClient(base_url=base_url, token=admin_key)
        token_endpoint = f"{base_url}/oauth/token"
        proof = prover.make_proof(htm="POST", htu=token_endpoint)
        tok_resp = requests.post(
            token_endpoint,
            data={
                "grant_type": "client_credentials",
                "client_id": client_id,
                "client_secret": client_secret,
                "scope": "vault:read",
            },
            headers={"DPoP": proof},
            timeout=10,
        )
        if tok_resp.status_code != 200:
            pytest.skip(
                f"Could not issue DPoP token for vault smoke: {tok_resp.text[:200]}"
            )

        bearer_token = tok_resp.json()["access_token"]

        vault = VaultClient(base_url=base_url, admin_key=admin_key)
        result = vault.fetch_token(
            provider=provider_slug,
            bearer_token=bearer_token,
            prover=prover,
        )

        assert isinstance(result, VaultTokenResult)
        assert result.access_token, "access_token must be non-empty"
        assert result.token_type, "token_type must be non-empty"

    except Exception as exc:
        # If vault is not wired, skip gracefully.
        if "404" in str(exc) or "not found" in str(exc).lower():
            pytest.skip(f"Vault not provisioned for provider '{provider_slug}': {exc}")
        raise

    finally:
        try:
            agents.revoke_agent(agent_id)
        except Exception:
            pass
