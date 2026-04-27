"""Smoke tests for MayActClient — mock-based, no live server."""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

from shark_auth.may_act import MayActClient


BASE = "https://auth.example.com"
TOKEN = "sk_live_test"


def _mock_resp(status: int, body: object):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = body
    resp.text = json.dumps(body)
    return resp


def test_find_builds_query_and_unwraps_grants():
    """find() must build the right query string and return the inner array."""
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(
            200,
            {"grants": [{"id": "mag_a", "from_id": "agent-a", "to_id": "u-1"}]},
        )
        client = MayActClient(base_url=BASE, admin_api_key=TOKEN)
        out = client.find(from_id="agent-a", include_revoked=True)

    # Response unwrapped from {grants: [...]}
    assert isinstance(out, list)
    assert out == [{"id": "mag_a", "from_id": "agent-a", "to_id": "u-1"}]

    # Query params correctly forwarded
    _, kwargs = mock_req.call_args[0], mock_req.call_args[1]
    params = kwargs["params"]
    assert params["from_id"] == "agent-a"
    assert params["include_revoked"] == "true"
    assert "to_id" not in params


def test_find_default_omits_include_revoked():
    with patch("shark_auth._http.request") as mock_req:
        mock_req.return_value = _mock_resp(200, {"grants": []})
        MayActClient(base_url=BASE, admin_api_key=TOKEN).find(to_id="u-9")
    _, kwargs = mock_req.call_args[0], mock_req.call_args[1]
    params = kwargs["params"]
    assert params == {"to_id": "u-9"}
