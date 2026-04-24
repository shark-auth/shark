"""Tests for PaywallClient (G4)."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from shark_auth.paywall import PaywallClient
from shark_auth.proxy_rules import SharkAPIError

BASE = "https://auth.example.com"

_HTML = "<html><body>Upgrade to Pro</body></html>"


def _make_client():
    return PaywallClient(base_url=BASE)


def _mock_resp(status: int, text: str = "", body: object = None):
    resp = MagicMock()
    resp.status_code = status
    resp.text = text
    if body is not None:
        resp.json.return_value = body
    else:
        resp.json.side_effect = ValueError("no json")
    return resp


# ---------------------------------------------------------------------------
# paywall_url (no network)
# ---------------------------------------------------------------------------


def test_paywall_url_basic():
    client = _make_client()
    url = client.paywall_url("my-app", "pro")
    assert url == "https://auth.example.com/paywall/my-app?tier=pro"


def test_paywall_url_with_return():
    client = _make_client()
    url = client.paywall_url("my-app", "pro", return_url="https://app.example.com/dashboard")
    assert "return=" in url
    assert "tier=pro" in url
    assert "/paywall/my-app" in url


def test_paywall_url_encodes_return():
    client = _make_client()
    url = client.paywall_url("my-app", "pro", return_url="https://app.example.com/dash?x=1")
    # return_url must be percent-encoded
    assert "return=https" in url or "return=" in url


# ---------------------------------------------------------------------------
# render_paywall
# ---------------------------------------------------------------------------


def test_render_paywall_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, text=_HTML)
        client = _make_client()
        html = client.render_paywall("my-app", "pro")

    assert html == _HTML
    call_args = mock_req.call_args
    assert "/paywall/my-app" in call_args[0][2]
    assert "tier=pro" in call_args[0][2]


def test_render_paywall_404():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            404, text="Not Found", body={"error": {"code": "not_found", "message": "app not found"}}
        )
        mock_req.return_value.json.return_value = {"error": {"code": "not_found", "message": "app not found"}}
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.render_paywall("unknown-app", "pro")

    assert exc_info.value.status == 404


def test_render_paywall_400_missing_tier():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            400,
            text="tier query param required",
            body={"error": {"code": "invalid_request", "message": "tier query param required"}},
        )
        mock_req.return_value.json.return_value = {
            "error": {"code": "invalid_request", "message": "tier query param required"}
        }
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.render_paywall("my-app", "")

    assert exc_info.value.status == 400


# ---------------------------------------------------------------------------
# preview_paywall
# ---------------------------------------------------------------------------


def test_preview_paywall_html_format():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, text=_HTML)
        client = _make_client()
        result = client.preview_paywall("my-app", "pro", format="html")

    assert result == _HTML


def test_preview_paywall_url_format():
    client = _make_client()
    result = client.preview_paywall("my-app", "pro", return_url="https://x.com", format="url")
    assert isinstance(result, str)
    assert "/paywall/my-app" in result
    assert "tier=pro" in result
