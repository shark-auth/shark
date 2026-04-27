"""Tests for AgentsClient (G6)."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from shark_auth.agents import AgentsClient
from shark_auth.proxy_rules import SharkAPIError

BASE = "https://auth.example.com"
TOKEN = "sk_live_test"

_AGENT = {
    "id": "agent_abc",
    "name": "my-bot",
    "client_id": "shark_agent_xyz",
    "active": True,
    "scopes": ["read"],
}

_AGENT_WITH_SECRET = {**_AGENT, "client_secret": "s3cr3t_once"}


def _make_client():
    return AgentsClient(base_url=BASE, token=TOKEN)


def _mock_resp(status: int, body: object):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = body
    resp.text = json.dumps(body)
    return resp


# ---------------------------------------------------------------------------
# register_agent
# ---------------------------------------------------------------------------


def test_register_agent_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(201, _AGENT_WITH_SECRET)
        client = _make_client()
        agent = client.register_agent(app_id="app_abc", name="my-bot", scopes=["read"])

    assert agent["id"] == "agent_abc"
    assert agent["client_secret"] == "s3cr3t_once"
    call_args = mock_req.call_args
    assert call_args[0][1] == "POST"
    assert call_args[0][2].endswith("/api/v1/agents")
    body = call_args[1]["json"]
    assert body["name"] == "my-bot"
    assert body["scopes"] == ["read"]
    assert body["metadata"]["app_id"] == "app_abc"


def test_register_agent_no_scopes():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(201, _AGENT_WITH_SECRET)
        client = _make_client()
        client.register_agent(app_id="app_abc", name="bot")

    body = mock_req.call_args[1]["json"]
    assert body["scopes"] == []


def test_register_agent_401():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            401, {"error": {"code": "unauthorized", "message": "bad key"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.register_agent("app_abc", "bot")

    assert exc_info.value.status == 401


def test_register_agent_403():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            403, {"error": {"code": "forbidden", "message": "forbidden"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.register_agent("app_abc", "bot")

    assert exc_info.value.status == 403


def test_register_agent_400_missing_name():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            400, {"error": {"code": "invalid_request", "message": "name is required"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.register_agent("app_abc", "")

    assert exc_info.value.code == "invalid_request"


# ---------------------------------------------------------------------------
# list_agents
# ---------------------------------------------------------------------------


def test_list_agents_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": [_AGENT], "total": 1})
        client = _make_client()
        agents = client.list_agents()

    assert len(agents) == 1
    assert agents[0]["id"] == "agent_abc"


def test_list_agents_filter_by_app_id():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": []})
        client = _make_client()
        client.list_agents(app_id="app_abc")

    params = mock_req.call_args[1].get("params", {})
    assert "app_abc" in params.get("search", "")


def test_list_agents_401():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            401, {"error": {"code": "unauthorized", "message": "bad key"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.list_agents()

    assert exc_info.value.status == 401


# ---------------------------------------------------------------------------
# revoke_agent
# ---------------------------------------------------------------------------


def test_revoke_agent_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(204, None)
        mock_req.return_value.text = ""
        client = _make_client()
        result = client.revoke_agent("agent_abc")

    assert result is None
    call_args = mock_req.call_args
    assert call_args[0][1] == "DELETE"
    assert "/agents/agent_abc" in call_args[0][2]


def test_revoke_agent_404():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            404, {"error": {"code": "not_found", "message": "agent not found"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.revoke_agent("nonexistent")

    assert exc_info.value.status == 404


def test_revoke_agent_403():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            403, {"error": {"code": "forbidden", "message": "forbidden"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.revoke_agent("agent_abc")

    assert exc_info.value.status == 403


# ---------------------------------------------------------------------------
# P1 smoke tests — list_tokens, revoke_all, rotate_secret, rotate_dpop_key
# ---------------------------------------------------------------------------


def test_list_tokens_hits_correct_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": [{"id": "tok_1", "agent_id": "agent_abc"}]})
        client = _make_client()
        tokens = client.list_tokens("agent_abc")
    assert mock_req.call_args[0][1] == "GET"
    assert mock_req.call_args[0][2].endswith("/api/v1/agents/agent_abc/tokens")
    assert tokens[0].token_id == "tok_1"


def test_revoke_all_hits_correct_path():
    with patch("shark_auth._http.request") as mock_req:
        resp = _mock_resp(200, {"revoked_count": 3})
        resp.content = b'{"revoked_count": 3}'
        mock_req.return_value = resp
        client = _make_client()
        result = client.revoke_all("agent_abc")
    assert mock_req.call_args[0][1] == "POST"
    assert mock_req.call_args[0][2].endswith("/api/v1/agents/agent_abc/tokens/revoke-all")
    assert result.revoked_count == 3


def test_rotate_secret_hits_correct_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            200, {"agent_id": "agent_abc", "client_id": "cid", "client_secret": "s3cr3t"}
        )
        client = _make_client()
        creds = client.rotate_secret("agent_abc")
    assert mock_req.call_args[0][1] == "POST"
    assert mock_req.call_args[0][2].endswith("/api/v1/agents/agent_abc/rotate-secret")
    assert creds.client_secret == "s3cr3t"


def test_rotate_dpop_key_hits_correct_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            200, {"old_jkt": "old", "new_jkt": "new", "revoked_token_count": 2, "audit_event_id": "ev_1"}
        )
        client = _make_client()
        result = client.rotate_dpop_key("agent_abc", new_public_key_jwk={"kty": "EC"})
    assert mock_req.call_args[0][1] == "POST"
    assert mock_req.call_args[0][2].endswith("/api/v1/agents/agent_abc/rotate-dpop-key")
    assert result.new_jkt == "new"
