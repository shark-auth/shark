"""Tests for UsersClient (G5)."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from shark_auth.proxy_rules import SharkAPIError
from shark_auth.users import UsersClient

BASE = "https://auth.example.com"
TOKEN = "sk_live_test"

_USER = {"id": "usr_abc", "email": "alice@example.com", "metadata": {"tier": "free"}}


def _make_client():
    return UsersClient(base_url=BASE, token=TOKEN)


def _mock_resp(status: int, body: object):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = body
    resp.text = json.dumps(body)
    return resp


# ---------------------------------------------------------------------------
# list_users
# ---------------------------------------------------------------------------


def test_list_users_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": [_USER], "total": 1})
        client = _make_client()
        users = client.list_users()

    assert len(users) == 1
    assert users[0]["id"] == "usr_abc"


def test_list_users_filter_by_email():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": [_USER]})
        client = _make_client()
        users = client.list_users(email="alice@example.com")

    call_args = mock_req.call_args
    params = call_args[1].get("params", {})
    assert params.get("email") == "alice@example.com"


def test_list_users_401():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            401, {"error": {"code": "unauthorized", "message": "bad key"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.list_users()

    assert exc_info.value.status == 401


# ---------------------------------------------------------------------------
# get_user
# ---------------------------------------------------------------------------


def test_get_user_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _USER})
        client = _make_client()
        user = client.get_user("usr_abc")

    assert user["email"] == "alice@example.com"
    assert "/users/usr_abc" in mock_req.call_args[0][2]


def test_get_user_404():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            404, {"error": {"code": "not_found", "message": "user not found"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.get_user("nonexistent")

    assert exc_info.value.status == 404
    assert exc_info.value.code == "not_found"


# ---------------------------------------------------------------------------
# set_user_tier
# ---------------------------------------------------------------------------


def test_set_user_tier_to_pro():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": {"user": _USER, "tier": "pro"}})
        client = _make_client()
        result = client.set_user_tier("usr_abc", "pro")

    assert result["tier"] == "pro"
    call_args = mock_req.call_args
    assert call_args[0][1] == "PATCH"
    assert "/admin/users/usr_abc/tier" in call_args[0][2]
    assert call_args[1]["json"]["tier"] == "pro"


def test_set_user_tier_to_free():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": {"user": _USER, "tier": "free"}})
        client = _make_client()
        result = client.set_user_tier("usr_abc", "free")

    assert result["tier"] == "free"


def test_set_user_tier_invalid_tier_400():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            400, {"error": {"code": "invalid_tier", "message": 'tier must be "free" or "pro"'}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.set_user_tier("usr_abc", "enterprise")  # type: ignore[arg-type]

    assert exc_info.value.code == "invalid_tier"
    assert exc_info.value.status == 400


def test_set_user_tier_user_not_found_404():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            404, {"error": {"code": "not_found", "message": "user not found"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.set_user_tier("nonexistent", "pro")

    assert exc_info.value.status == 404


def test_set_user_tier_403():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            403, {"error": {"code": "forbidden", "message": "forbidden"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.set_user_tier("usr_abc", "pro")

    assert exc_info.value.status == 403
