"""Tests for ProxyRulesClient (G1)."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from shark_auth.proxy_rules import (
    CreateProxyRuleInput,
    ImportResult,
    ProxyRule,
    ProxyRulesClient,
    SharkAPIError,
)

BASE = "https://auth.example.com"
TOKEN = "sk_live_test"


def _make_client():
    return ProxyRulesClient(base_url=BASE, token=TOKEN)


def _mock_resp(status: int, body: object):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = body
    resp.text = json.dumps(body)
    return resp


# ---------------------------------------------------------------------------
# list_rules
# ---------------------------------------------------------------------------


def test_list_rules_happy_path():
    rule = {"id": "rule_1", "name": "test", "pattern": "/api/*", "enabled": True}
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": [rule], "total": 1})
        client = _make_client()
        rules = client.list_rules()

    assert len(rules) == 1
    assert rules[0]["id"] == "rule_1"
    # Verify the request was actually made
    mock_req.assert_called_once()


def test_list_rules_with_app_id():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": [], "total": 0})
        client = _make_client()
        client.list_rules(app_id="app_abc")

    # Verify the request was called with app_id param
    call_args = mock_req.call_args
    params = call_args[1].get("params") or {}
    assert params.get("app_id") == "app_abc"


def test_list_rules_401():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            401, {"error": {"code": "unauthorized", "message": "missing admin key"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.list_rules()

    assert exc_info.value.status == 401
    assert exc_info.value.code == "unauthorized"


def test_list_rules_403():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            403, {"error": {"code": "forbidden", "message": "insufficient permissions"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.list_rules()

    assert exc_info.value.status == 403


# ---------------------------------------------------------------------------
# create_rule
# ---------------------------------------------------------------------------


def test_create_rule_happy_path():
    rule = {
        "id": "rule_new",
        "name": "block-write",
        "pattern": "/api/write/*",
        "enabled": True,
        "priority": 100,
    }
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(201, {"data": rule})
        client = _make_client()
        created = client.create_rule(
            CreateProxyRuleInput(name="block-write", pattern="/api/write/*")  # type: ignore[call-arg]
        )

    assert created["id"] == "rule_new"


def test_create_rule_422():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            400, {"error": {"code": "invalid_proxy_rule", "message": "pattern must start with '/'"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.create_rule(CreateProxyRuleInput(name="x", pattern="bad"))  # type: ignore[call-arg]

    assert exc_info.value.code == "invalid_proxy_rule"
    assert exc_info.value.status == 400


# ---------------------------------------------------------------------------
# get_rule
# ---------------------------------------------------------------------------


def test_get_rule_happy_path():
    rule = {"id": "rule_1", "name": "test", "pattern": "/api/*"}
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": rule})
        client = _make_client()
        got = client.get_rule("rule_1")

    assert got["id"] == "rule_1"


def test_get_rule_404():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            404, {"error": {"code": "not_found", "message": "rule not found"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.get_rule("nonexistent")

    assert exc_info.value.status == 404


# ---------------------------------------------------------------------------
# update_rule
# ---------------------------------------------------------------------------


def test_update_rule_happy_path():
    rule = {"id": "rule_1", "enabled": False}
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": rule})
        client = _make_client()
        updated = client.update_rule("rule_1", enabled=False)

    assert updated["enabled"] is False


# ---------------------------------------------------------------------------
# delete_rule
# ---------------------------------------------------------------------------


def test_delete_rule_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(204, None)
        mock_req.return_value.text = ""
        client = _make_client()
        result = client.delete_rule("rule_1")

    assert result is None


def test_delete_rule_404():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            404, {"error": {"code": "not_found", "message": "not found"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.delete_rule("nope")

    assert exc_info.value.status == 404


# ---------------------------------------------------------------------------
# Composition test — Client builds correct URL + auth header
# ---------------------------------------------------------------------------


def test_client_composition_builds_correct_url_and_header():
    """Verify Client(token=...).proxy_rules.list_rules() sends correct URL + header."""
    from shark_auth import Client

    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": [], "total": 0})
        c = Client(base_url="https://auth.example.com", token="sk_live_xyz")
        c.proxy_rules.list_rules()

    call_args = mock_req.call_args
    # positional: (session, method, url, ...)
    assert call_args[0][1] == "GET"
    assert call_args[0][2] == "https://auth.example.com/api/v1/admin/proxy/rules/db"
    headers = call_args[1].get("headers") or call_args[0][3]
    assert headers["Authorization"] == "Bearer sk_live_xyz"
