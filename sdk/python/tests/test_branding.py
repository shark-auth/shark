"""Tests for BrandingClient (G3)."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

from shark_auth.branding import BrandingClient
from shark_auth.proxy_rules import SharkAPIError

BASE = "https://auth.example.com"
TOKEN = "sk_live_test"

_BRANDING_ROW = {
    "id": "branding_global",
    "primary_color": "#6366f1",
    "font": "Inter",
}

_DESIGN_TOKENS = {
    "colors": {"primary": "#6366f1"},
    "typography": {"font_family": "Inter"},
}


def _make_client():
    return BrandingClient(base_url=BASE, token=TOKEN)


def _mock_resp(status: int, body: object):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = body
    resp.text = json.dumps(body)
    return resp


# ---------------------------------------------------------------------------
# get_branding
# ---------------------------------------------------------------------------


def test_get_branding_happy_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _BRANDING_ROW})
        client = _make_client()
        branding = client.get_branding()

    assert branding["id"] == "branding_global"
    assert mock_req.call_args[0][1] == "GET"
    assert "/admin/branding" in mock_req.call_args[0][2]


def test_get_branding_with_app_slug():
    """app_slug is accepted for symmetry — still calls the same endpoint."""
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"data": _BRANDING_ROW})
        client = _make_client()
        branding = client.get_branding(app_slug="my-app")

    assert branding["id"] == "branding_global"


def test_get_branding_401():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            401, {"error": {"code": "unauthorized", "message": "missing key"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.get_branding()

    assert exc_info.value.status == 401


# ---------------------------------------------------------------------------
# set_branding
# ---------------------------------------------------------------------------


def test_set_branding_happy_path():
    resp_body = {
        "data": {
            "branding": _BRANDING_ROW,
            "design_tokens": _DESIGN_TOKENS,
        }
    }
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, resp_body)
        client = _make_client()
        result = client.set_branding(tokens=_DESIGN_TOKENS)

    assert result["design_tokens"]["colors"]["primary"] == "#6366f1"
    call_args = mock_req.call_args
    assert call_args[0][1] == "PATCH"
    assert "/branding/design-tokens" in call_args[0][2]
    payload = call_args[1]["json"]
    assert payload["design_tokens"] == _DESIGN_TOKENS


def test_set_branding_empty_tokens():
    resp_body = {"data": {"branding": _BRANDING_ROW, "design_tokens": {}}}
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, resp_body)
        client = _make_client()
        result = client.set_branding()

    assert result["design_tokens"] == {}


def test_set_branding_401():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            401, {"error": {"code": "unauthorized", "message": "bad key"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.set_branding(tokens={"x": 1})

    assert exc_info.value.status == 401


def test_set_branding_400():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            400, {"error": {"code": "invalid_request", "message": "Invalid JSON body"}}
        )
        client = _make_client()
        with pytest.raises(SharkAPIError) as exc_info:
            client.set_branding(tokens={"x": 1})

    assert exc_info.value.code == "invalid_request"
