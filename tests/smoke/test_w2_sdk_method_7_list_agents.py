"""Smoke — Method 7: UsersClient.list_agents() — filter=created/authorized/all.

Test flow: create a user → register an agent under that user →
list_agents with each filter value → assert shape.
"""

from __future__ import annotations

import pytest

from shark_auth import Client
from shark_auth.users import AgentList

BASE_URL = "http://localhost:8080"


@pytest.fixture(scope="module")
def client(admin_key):
    return Client(base_url=BASE_URL, token=admin_key)


@pytest.fixture(scope="module")
def list_user(client):
    """Create a fresh user for list_agents tests."""
    user = client.users.create_user(
        "list-agents-smoke@example.com",
        name="List Agents Smoke User",
    )
    yield user
    try:
        client.users.delete_user(user["id"])
    except Exception:
        pass


@pytest.fixture(scope="module")
def list_agent(client, list_user):
    """Register one agent for the list_agents smoke user."""
    agent = client.agents.register_agent(
        app_id="smoke-m7",
        name="list-agents-smoke-agent",
        scopes=["mcp:read"],
    )
    yield agent
    try:
        client.agents.revoke_agent(agent["id"])
    except Exception:
        pass


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

class TestListAgentsReturnType:
    def test_returns_agent_list(self, client, list_user, list_agent):
        result = client.users.list_agents(list_user["id"])
        assert isinstance(result, AgentList)

    def test_data_is_list(self, client, list_user, list_agent):
        result = client.users.list_agents(list_user["id"])
        assert isinstance(result.data, list)

    def test_total_is_int(self, client, list_user, list_agent):
        result = client.users.list_agents(list_user["id"])
        assert isinstance(result.total, int)
        assert result.total >= 0


class TestListAgentsFilters:
    def test_filter_created(self, client, list_user, list_agent):
        """filter='created' should not raise and return AgentList."""
        result = client.users.list_agents(list_user["id"], filter="created")
        assert isinstance(result, AgentList)
        assert isinstance(result.data, list)

    def test_filter_authorized(self, client, list_user, list_agent):
        """filter='authorized' should not raise and return AgentList."""
        result = client.users.list_agents(list_user["id"], filter="authorized")
        assert isinstance(result, AgentList)
        assert isinstance(result.data, list)

    def test_filter_all(self, client, list_user, list_agent):
        """filter='all' should not raise and return AgentList."""
        result = client.users.list_agents(list_user["id"], filter="all")
        assert isinstance(result, AgentList)
        assert isinstance(result.data, list)


class TestListAgentsPagination:
    def test_limit_param(self, client, list_user, list_agent):
        result = client.users.list_agents(list_user["id"], limit=1)
        assert isinstance(result, AgentList)
        assert len(result.data) <= 1

    def test_offset_param(self, client, list_user, list_agent):
        result = client.users.list_agents(list_user["id"], offset=0, limit=10)
        assert isinstance(result, AgentList)
