"""Tests for ProxyLifecycleClient (G2)."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from shark_auth.proxy_lifecycle import ProxyLifecycleClient, ProxyStatus
from shark_auth.proxy_rules import SharkAPIError

BASE = "https://auth.example.com"
TOKEN = "sk_live_test"

_RUNNING_STATUS = {
    "state": 1,
    "state_str": "running",
    "listeners": 2,
    "rules_loaded": 5,
    "started_at": "2026-04-24T10:00:00Z",
    "last_error": "",
}

_STOPPED_STATUS = {
    "state": 0,
    "state_str": "stopped",
    "listeners": 0,
    "rules_loaded": 0,
    "started_at": "",
    "last_error": "",
}


def _make_client():
    return ProxyLifecycleClient(base_url=BASE, token=TOKEN)


def _mock_resp(status: int, body: object):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = body
    resp.text = json.dumps(body)
    return resp


# ---------------------------------------------------------------------------
# get_proxy_status
# ---------------------------------------------------------------------------


def test_get_proxy_status_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _RUNNING_STATUS})
        client = _make_client()
        status = client.get_proxy_status()

    assert status["state_str"] == "running"
    assert status["listeners"] == 2

    call_args = mock_req.call_args
    assert call_args[0][1] == "GET"
    assert "/proxy/lifecycle" in call_args[0][2]


def test_get_proxy_status_401():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            401, {"error": {"code": "unauthorized", "message": "missing key"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.get_proxy_status()

    assert exc_info.value.status == 401


# ---------------------------------------------------------------------------
# start_proxy
# ---------------------------------------------------------------------------


def test_start_proxy_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _RUNNING_STATUS})
        client = _make_client()
        status = client.start_proxy()

    assert status["state_str"] == "running"
    assert mock_req.call_args[0][1] == "POST"
    assert "/proxy/start" in mock_req.call_args[0][2]


def test_start_proxy_409_conflict():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            409, {"error": {"code": "proxy_start_failed", "message": "already running"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.start_proxy()

    assert exc_info.value.status == 409
    assert exc_info.value.code == "proxy_start_failed"


# ---------------------------------------------------------------------------
# stop_proxy
# ---------------------------------------------------------------------------


def test_stop_proxy_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _STOPPED_STATUS})
        client = _make_client()
        status = client.stop_proxy()

    assert status["state_str"] == "stopped"
    assert "/proxy/stop" in mock_req.call_args[0][2]


def test_stop_proxy_idempotent():
    """Stopping a stopped proxy should return 200 per spec."""
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _STOPPED_STATUS})
        client = _make_client()
        status = client.stop_proxy()

    assert status["state_str"] == "stopped"


# ---------------------------------------------------------------------------
# reload_proxy
# ---------------------------------------------------------------------------


def test_reload_proxy_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _RUNNING_STATUS})
        client = _make_client()
        status = client.reload_proxy()

    assert status["state_str"] == "running"
    assert "/proxy/reload" in mock_req.call_args[0][2]


def test_reload_proxy_403():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            403, {"error": {"code": "forbidden", "message": "forbidden"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.reload_proxy()

    assert exc_info.value.status == 403
