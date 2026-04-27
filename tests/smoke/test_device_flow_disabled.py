"""Smoke tests asserting device flow is correctly disabled for v0.1.

These tests MUST PASS (not xfail) — they verify the disable holds:
  - AS metadata does NOT advertise device_authorization_endpoint
  - AS metadata grant_types_supported does NOT include device_code
  - POST /oauth/device returns 501 Not Implemented
  - POST /oauth/token with device_code grant_type returns unsupported_grant_type
"""
import pytest
import requests
import os

BASE_URL = os.environ.get("BASE", "http://localhost:8080")

DEVICE_GRANT = "urn:ietf:params:oauth:grant-type:device_code"


class TestDeviceFlowDisabled:
    def test_as_metadata_no_device_authorization_endpoint(self):
        """AS metadata must NOT contain device_authorization_endpoint."""
        resp = requests.get(f"{BASE_URL}/.well-known/oauth-authorization-server")
        assert resp.status_code == 200, f"Metadata endpoint failed: {resp.text}"
        meta = resp.json()
        assert "device_authorization_endpoint" not in meta, (
            f"device_authorization_endpoint should be absent from AS metadata for v0.1, "
            f"got: {meta.get('device_authorization_endpoint')}"
        )

    def test_as_metadata_no_device_code_grant_type(self):
        """AS metadata grant_types_supported must NOT include device_code grant."""
        resp = requests.get(f"{BASE_URL}/.well-known/oauth-authorization-server")
        assert resp.status_code == 200, f"Metadata endpoint failed: {resp.text}"
        meta = resp.json()
        grant_types = meta.get("grant_types_supported", [])
        assert DEVICE_GRANT not in grant_types, (
            f"device_code grant should be absent from grant_types_supported for v0.1, "
            f"got: {grant_types}"
        )

    def test_device_authorization_endpoint_returns_501(self):
        """POST /oauth/device must return 501 Not Implemented."""
        resp = requests.post(
            f"{BASE_URL}/oauth/device",
            data={"client_id": "any-client", "scope": "read"},
        )
        assert resp.status_code == 501, (
            f"Expected 501 from /oauth/device (device flow disabled), got {resp.status_code}: {resp.text}"
        )
        body = resp.json()
        assert body.get("error") == "not_implemented", (
            f"Expected error=not_implemented, got: {body}"
        )

    def test_token_endpoint_device_code_grant_returns_unsupported(self):
        """POST /oauth/token with device_code grant must return unsupported_grant_type."""
        resp = requests.post(
            f"{BASE_URL}/oauth/token",
            data={
                "grant_type": DEVICE_GRANT,
                "device_code": "fake-device-code-for-disabled-check",
                "client_id": "any-client",
            },
        )
        assert resp.status_code == 501, (
            f"Expected 501 from /oauth/token with device_code grant (disabled), "
            f"got {resp.status_code}: {resp.text}"
        )
        body = resp.json()
        assert body.get("error") == "unsupported_grant_type", (
            f"Expected error=unsupported_grant_type, got: {body}"
        )

    def test_device_verify_page_returns_501(self):
        """GET /oauth/device/verify must return 501 Not Implemented."""
        resp = requests.get(f"{BASE_URL}/oauth/device/verify")
        assert resp.status_code == 501, (
            f"Expected 501 from /oauth/device/verify (device flow disabled), "
            f"got {resp.status_code}"
        )
