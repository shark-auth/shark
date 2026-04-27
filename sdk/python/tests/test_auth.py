"""Tests for AuthClient — P2 additions: check() and revoke_self()."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

from shark_auth.auth import AuthClient

BASE = "https://auth.example.com"


def _make_client():
    return AuthClient(BASE)


def _mock_resp(status: int, body: object):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = body
    resp.text = json.dumps(body)
    return resp


# ---------------------------------------------------------------------------
# P2 smoke tests — check, revoke_self
# ---------------------------------------------------------------------------


def test_check_hits_correct_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"allowed": True})
        client = _make_client()
        result = client.check("read", "documents:123")
    assert mock_req.call_args[0][1] == "POST"
    assert mock_req.call_args[0][2].endswith("/api/v1/auth/check")
    body = mock_req.call_args[1]["json"]
    assert body == {"action": "read", "resource": "documents:123"}
    assert result["allowed"] is True


def test_revoke_self_hits_correct_path():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(204, {})
        mock_req.return_value.status_code = 204
        client = _make_client()
        result = client.revoke_self()
    assert result is None
    assert mock_req.call_args[0][1] == "POST"
    assert mock_req.call_args[0][2].endswith("/api/v1/auth/revoke")
