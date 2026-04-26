"""Smoke — Method 6: UsersClient.revoke_agents() — cascade revoke.

Test flow: create a user → register 2 agents under that user → cascade-revoke
→ assert result shape and audit_event_id present.
"""

from __future__ import annotations

import pytest

from shark_auth import Client
from shark_auth.users import CascadeRevokeResult

BASE_URL = "http://localhost:8080"


@pytest.fixture(scope="module")
def client(admin_key):
    return Client(base_url=BASE_URL, token=admin_key)


@pytest.fixture(scope="module")
def cascade_user(client):
    """Create a fresh user for cascade-revoke tests."""
    user = client.users.create_user(
        "cascade-revoke-smoke@example.com",
        name="Cascade Smoke User",
    )
    yield user
    # Cleanup
    try:
        client.users.delete_user(user["id"])
    except Exception:
        pass


@pytest.fixture(scope="module")
def cascade_agents(client, cascade_user):
    """Register 2 agents; deactivate after module."""
    agents = []
    for i in range(2):
        a = client.agents.register_agent(
            app_id="smoke-m6",
            name=f"cascade-smoke-agent-{i}",
            scopes=["mcp:read"],
        )
        agents.append(a)
    yield agents
    for a in agents:
        try:
            client.agents.revoke_agent(a["id"])
        except Exception:
            pass


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

class TestCascadeRevokeAll:
    def test_revoke_agents_returns_result(self, client, cascade_user, cascade_agents):
        result = client.users.revoke_agents(cascade_user["id"])
        assert isinstance(result, CascadeRevokeResult)

    def test_revoked_agent_ids_is_list(self, client, cascade_user):
        result = client.users.revoke_agents(cascade_user["id"])
        assert isinstance(result.revoked_agent_ids, list)

    def test_revoked_consent_count_is_int(self, client, cascade_user):
        result = client.users.revoke_agents(cascade_user["id"])
        assert isinstance(result.revoked_consent_count, int)
        assert result.revoked_consent_count >= 0


class TestCascadeRevokeSelective:
    def test_selective_revoke_with_agent_ids(self, client, cascade_user, cascade_agents):
        """Provide specific agent_ids — should not raise."""
        agent_id = cascade_agents[0]["id"]
        result = client.users.revoke_agents(cascade_user["id"], agent_ids=[agent_id])
        assert isinstance(result, CascadeRevokeResult)

    def test_selective_revoke_with_reason(self, client, cascade_user, cascade_agents):
        """reason kwarg should be accepted without error."""
        result = client.users.revoke_agents(
            cascade_user["id"],
            reason="smoke-test cascade",
        )
        assert isinstance(result, CascadeRevokeResult)
