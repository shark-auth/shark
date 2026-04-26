"""
Wave 1.5 – Bug B smoke tests
DELETE /api/v1/users/{id} must revoke all agent tokens and sessions before deletion.
"""
import pytest
import requests

from conftest import base_url, admin_headers, register_user, register_agent, issue_token


def test_delete_user_revokes_agent_tokens_and_sessions():
    """Happy path: deleting a user revokes agent tokens and sessions."""
    # Register user, agent, and issue a token
    user = register_user()
    user_id = user["id"]
    agent = register_agent(user_id)
    client_id = agent["client_id"]
    client_secret = agent["client_secret"]

    token = issue_token(client_id, client_secret)
    assert token, "Expected a valid token before user deletion"

    # Confirm token is active
    introspect_before = requests.post(
        f"{base_url()}/oauth/introspect",
        data={"token": token},
        headers=admin_headers(),
    )
    assert introspect_before.status_code == 200
    assert introspect_before.json().get("active") is True

    # Delete the user
    del_resp = requests.delete(
        f"{base_url()}/api/v1/users/{user_id}",
        headers=admin_headers(),
    )
    assert del_resp.status_code == 200, f"DELETE user failed: {del_resp.text}"

    # Token issued for the user's agent should now be revoked
    introspect_after = requests.post(
        f"{base_url()}/oauth/introspect",
        data={"token": token},
        headers=admin_headers(),
    )
    assert introspect_after.status_code == 200
    assert introspect_after.json().get("active") is False, (
        "Agent token should be revoked after user deletion"
    )

    # The user should no longer exist
    get_resp = requests.get(
        f"{base_url()}/api/v1/users/{user_id}",
        headers=admin_headers(),
    )
    assert get_resp.status_code == 404


def test_delete_unknown_user_returns_404():
    """Negative: DELETE on an unknown user ID returns 404."""
    resp = requests.delete(
        f"{base_url()}/api/v1/users/user_does_not_exist",
        headers=admin_headers(),
    )
    assert resp.status_code == 404
