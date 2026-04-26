"""
Wave 1.5 – Bug A smoke tests
PATCH /api/v1/agents/{id} active=false must revoke all existing tokens.
"""
import pytest
import requests

from conftest import base_url, admin_headers, register_user, register_agent, issue_token


def test_disable_agent_revokes_existing_tokens():
    """Happy path: deactivating agent revokes its live tokens."""
    # Register a user and agent
    user = register_user()
    agent = register_agent(user["id"])
    client_id = agent["client_id"]
    client_secret = agent["client_secret"]

    # Issue a token for the agent
    token = issue_token(client_id, client_secret)
    assert token, "Expected a valid token"

    # Verify token is active via introspection
    resp = requests.post(
        f"{base_url()}/oauth/introspect",
        data={"token": token},
        headers=admin_headers(),
    )
    assert resp.status_code == 200
    assert resp.json().get("active") is True, "Token should be active before deactivation"

    # Deactivate the agent
    patch_resp = requests.patch(
        f"{base_url()}/api/v1/agents/{agent['id']}",
        json={"active": False},
        headers=admin_headers(),
    )
    assert patch_resp.status_code == 200, f"PATCH agent failed: {patch_resp.text}"
    assert patch_resp.json().get("active") is False

    # Verify the previously issued token is now revoked
    introspect_resp = requests.post(
        f"{base_url()}/oauth/introspect",
        data={"token": token},
        headers=admin_headers(),
    )
    assert introspect_resp.status_code == 200
    assert introspect_resp.json().get("active") is False, (
        "Token should be inactive after agent deactivation"
    )


def test_disable_unknown_agent_returns_404():
    """Negative: PATCH on an unknown agent ID returns 404."""
    resp = requests.patch(
        f"{base_url()}/api/v1/agents/agent_does_not_exist",
        json={"active": False},
        headers=admin_headers(),
    )
    assert resp.status_code == 404
