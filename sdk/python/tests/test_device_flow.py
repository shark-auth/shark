"""Tests for DeviceFlow (RFC 8628)."""

from __future__ import annotations

from itertools import count

import pytest
from pytest_httpserver import HTTPServer

from shark_auth import DeviceFlow
from shark_auth.errors import DeviceFlowError

pytestmark = pytest.mark.skip(reason="Device flow is deferred from launch scope")


def _fake_clock_and_sleeper():
    """Deterministic monotonic clock that advances on each sleep()."""
    t = [0.0]

    def clock() -> float:
        return t[0]

    def sleeper(s: float) -> None:
        t[0] += s

    return clock, sleeper, t


def test_begin_parses_response(httpserver: HTTPServer):
    httpserver.expect_request(
        "/oauth/device_authorization", method="POST"
    ).respond_with_json(
        {
            "device_code": "dc_123",
            "user_code": "WDJB-MJHT",
            "verification_uri": "https://auth.example/device",
            "verification_uri_complete": "https://auth.example/device?user_code=WDJB-MJHT",
            "expires_in": 600,
            "interval": 5,
        }
    )
    flow = DeviceFlow(auth_url=httpserver.url_for(""), client_id="agent_abc")
    init = flow.begin()
    assert init.device_code == "dc_123"
    assert init.user_code == "WDJB-MJHT"
    assert init.verification_uri == "https://auth.example/device"
    assert init.verification_uri_complete.endswith("WDJB-MJHT")
    assert init.interval == 5


def test_wait_for_approval_success_after_pending(httpserver: HTTPServer):
    httpserver.expect_request(
        "/oauth/device_authorization", method="POST"
    ).respond_with_json(
        {
            "device_code": "dc_xyz",
            "user_code": "AAAA-BBBB",
            "verification_uri": "https://auth.example/device",
            "expires_in": 600,
            "interval": 1,
        }
    )

    responses = iter(
        [
            (400, {"error": "authorization_pending"}),
            (400, {"error": "authorization_pending"}),
            (
                200,
                {
                    "access_token": "at_success",
                    "token_type": "Bearer",
                    "expires_in": 3600,
                    "refresh_token": "rt_1",
                    "scope": "resource:read",
                },
            ),
        ]
    )

    def handler(request):
        from werkzeug.wrappers import Response
        import json

        status, body = next(responses)
        return Response(json.dumps(body), status=status, mimetype="application/json")

    httpserver.expect_request("/oauth/token", method="POST").respond_with_handler(
        handler
    )

    clock, sleeper, _ = _fake_clock_and_sleeper()
    flow = DeviceFlow(auth_url=httpserver.url_for(""), client_id="agent_abc")
    flow.begin()
    tok = flow.wait_for_approval(timeout_s=60, clock=clock, sleeper=sleeper)

    assert tok.access_token == "at_success"
    assert tok.token_type == "Bearer"
    assert tok.refresh_token == "rt_1"
    assert tok.scope == "resource:read"


def test_slow_down_increases_interval(httpserver: HTTPServer):
    httpserver.expect_request(
        "/oauth/device_authorization", method="POST"
    ).respond_with_json(
        {
            "device_code": "dc_s",
            "user_code": "CODE-1234",
            "verification_uri": "https://auth.example/device",
            "expires_in": 600,
            "interval": 2,
        }
    )

    responses = iter(
        [
            (400, {"error": "slow_down"}),
            (200, {"access_token": "at_ok", "token_type": "Bearer"}),
        ]
    )

    def handler(request):
        from werkzeug.wrappers import Response
        import json

        status, body = next(responses)
        return Response(json.dumps(body), status=status, mimetype="application/json")

    httpserver.expect_request("/oauth/token", method="POST").respond_with_handler(
        handler
    )

    sleep_durations: list[float] = []
    t = [0.0]

    def clock() -> float:
        return t[0]

    def sleeper(s: float) -> None:
        sleep_durations.append(s)
        t[0] += s

    flow = DeviceFlow(auth_url=httpserver.url_for(""), client_id="agent_abc")
    flow.begin()
    flow.wait_for_approval(timeout_s=120, clock=clock, sleeper=sleeper)

    # After slow_down, interval should have become 2 + 5 = 7
    assert sleep_durations == [7]


def test_access_denied_raises(httpserver: HTTPServer):
    httpserver.expect_request(
        "/oauth/device_authorization", method="POST"
    ).respond_with_json(
        {
            "device_code": "dc_d",
            "user_code": "CD-1",
            "verification_uri": "https://auth.example/device",
            "expires_in": 600,
            "interval": 1,
        }
    )
    httpserver.expect_request("/oauth/token", method="POST").respond_with_json(
        {"error": "access_denied"}, status=400
    )

    clock, sleeper, _ = _fake_clock_and_sleeper()
    flow = DeviceFlow(auth_url=httpserver.url_for(""), client_id="agent_abc")
    flow.begin()
    with pytest.raises(DeviceFlowError, match="denied"):
        flow.wait_for_approval(timeout_s=60, clock=clock, sleeper=sleeper)


def test_expired_token_raises(httpserver: HTTPServer):
    httpserver.expect_request(
        "/oauth/device_authorization", method="POST"
    ).respond_with_json(
        {
            "device_code": "dc_e",
            "user_code": "CE-1",
            "verification_uri": "https://auth.example/device",
            "expires_in": 600,
            "interval": 1,
        }
    )
    httpserver.expect_request("/oauth/token", method="POST").respond_with_json(
        {"error": "expired_token"}, status=400
    )

    clock, sleeper, _ = _fake_clock_and_sleeper()
    flow = DeviceFlow(auth_url=httpserver.url_for(""), client_id="agent_abc")
    flow.begin()
    with pytest.raises(DeviceFlowError, match="expired"):
        flow.wait_for_approval(timeout_s=60, clock=clock, sleeper=sleeper)


def test_timeout_raises(httpserver: HTTPServer):
    httpserver.expect_request(
        "/oauth/device_authorization", method="POST"
    ).respond_with_json(
        {
            "device_code": "dc_t",
            "user_code": "CT-1",
            "verification_uri": "https://auth.example/device",
            "expires_in": 600,
            "interval": 5,
        }
    )
    httpserver.expect_request("/oauth/token", method="POST").respond_with_json(
        {"error": "authorization_pending"}, status=400
    )

    clock, sleeper, _ = _fake_clock_and_sleeper()
    flow = DeviceFlow(auth_url=httpserver.url_for(""), client_id="agent_abc")
    flow.begin()
    with pytest.raises(DeviceFlowError, match="timed out"):
        flow.wait_for_approval(timeout_s=3, clock=clock, sleeper=sleeper)


def test_wait_without_begin_raises():
    flow = DeviceFlow(auth_url="https://x.example", client_id="agent")
    with pytest.raises(DeviceFlowError, match="begin"):
        flow.wait_for_approval()
