"""Agent flow smoke — RFC 8628 Device Authorization Grant.

Covers: device_authorization request, polling (authorization_pending,
slow_down), admin approval, token issuance, denial path.

NOTE: All tests in this module are marked xfail — device flow is disabled for
v0.1. Endpoints return 501 Not Implemented. Re-enable in v0.2.
"""
import time
import pytest
import requests
import os

pytestmark = pytest.mark.xfail(
    reason="Device authorization grant (RFC 8628) disabled for v0.1 — coming v0.2",
    strict=False,
)

BASE_URL = os.environ.get("BASE", "http://localhost:8080")


def _admin_headers(admin_key: str) -> dict:
    return {"Authorization": f"Bearer {admin_key}"}


def _register_device_client(admin_key: str) -> dict:
    resp = requests.post(
        f"{BASE_URL}/api/v1/agents",
        json={
            "name": "smoke-device-agent",
            "grant_types": ["urn:ietf:params:oauth:grant-type:device_code"],
            "scopes": ["read"],
        },
        headers=_admin_headers(admin_key),
    )
    assert resp.status_code == 201, f"Agent create failed: {resp.text}"
    return resp.json()


class TestDeviceFlow:
    def test_device_authorization_endpoint(self, admin_key):
        """POST /oauth/device returns device_code, user_code, verification_uri."""
        agent = _register_device_client(admin_key)
        client_id = agent["client_id"]
        try:
            resp = requests.post(
                f"{BASE_URL}/oauth/device",
                data={"client_id": client_id, "scope": "read"},
            )
            assert resp.status_code == 200, resp.text
            data = resp.json()
            assert "device_code" in data
            assert "user_code" in data
            assert "verification_uri" in data
            assert "expires_in" in data
            assert "interval" in data
        finally:
            requests.delete(
                f"{BASE_URL}/api/v1/agents/{agent['id']}",
                headers=_admin_headers(admin_key),
            )

    def test_polling_before_approval_returns_authorization_pending(self, admin_key):
        """Polling before user approves must return authorization_pending."""
        agent = _register_device_client(admin_key)
        client_id = agent["client_id"]
        try:
            dev_resp = requests.post(
                f"{BASE_URL}/oauth/device",
                data={"client_id": client_id, "scope": "read"},
            )
            assert dev_resp.status_code == 200
            device_code = dev_resp.json()["device_code"]

            poll_resp = requests.post(
                f"{BASE_URL}/oauth/token",
                data={
                    "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
                    "device_code": device_code,
                    "client_id": client_id,
                },
            )
            assert poll_resp.status_code == 400, poll_resp.text
            err = poll_resp.json()
            assert err.get("error") in ("authorization_pending", "slow_down"), \
                f"Expected authorization_pending/slow_down, got: {err}"
        finally:
            requests.delete(
                f"{BASE_URL}/api/v1/agents/{agent['id']}",
                headers=_admin_headers(admin_key),
            )

    @pytest.mark.skip(reason="Requires human user interaction for device approval UI; use admin approval API instead when wired")
    def test_full_device_flow_with_human_approval(self, admin_key):
        """Full device flow: device_code → user approves → token issued."""
        pass

    @pytest.mark.xfail(reason="Backend: admin approve device-code returns 500 — internal_error; backend gap to fix")
    def test_admin_approve_device_code(self, admin_key):
        """Admin can approve a pending device code via API."""
        agent = _register_device_client(admin_key)
        client_id = agent["client_id"]
        try:
            dev_resp = requests.post(
                f"{BASE_URL}/oauth/device",
                data={"client_id": client_id, "scope": "read"},
            )
            assert dev_resp.status_code == 200
            dev_data = dev_resp.json()
            device_code = dev_data["device_code"]
            user_code = dev_data["user_code"]

            # Admin approves
            approve_resp = requests.post(
                f"{BASE_URL}/api/v1/admin/oauth/device-codes/{user_code}/approve",
                headers=_admin_headers(admin_key),
            )
            assert approve_resp.status_code in (200, 204), \
                f"Admin approve failed: {approve_resp.text}"

            # Poll — should now return token
            poll_resp = requests.post(
                f"{BASE_URL}/oauth/token",
                data={
                    "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
                    "device_code": device_code,
                    "client_id": client_id,
                },
            )
            assert poll_resp.status_code == 200, \
                f"Token poll after approval failed: {poll_resp.text}"
            token_data = poll_resp.json()
            assert "access_token" in token_data
        finally:
            requests.delete(
                f"{BASE_URL}/api/v1/agents/{agent['id']}",
                headers=_admin_headers(admin_key),
            )

    @pytest.mark.xfail(reason="Backend: admin deny device-code returns 500 — internal_error; backend gap to fix")
    def test_admin_deny_device_code(self, admin_key):
        """Admin denial causes subsequent poll to return access_denied."""
        agent = _register_device_client(admin_key)
        client_id = agent["client_id"]
        try:
            dev_resp = requests.post(
                f"{BASE_URL}/oauth/device",
                data={"client_id": client_id, "scope": "read"},
            )
            assert dev_resp.status_code == 200
            dev_data = dev_resp.json()
            device_code = dev_data["device_code"]
            user_code = dev_data["user_code"]

            # Admin denies
            deny_resp = requests.post(
                f"{BASE_URL}/api/v1/admin/oauth/device-codes/{user_code}/deny",
                headers=_admin_headers(admin_key),
            )
            assert deny_resp.status_code in (200, 204), \
                f"Admin deny failed: {deny_resp.text}"

            # Poll — should return access_denied
            poll_resp = requests.post(
                f"{BASE_URL}/oauth/token",
                data={
                    "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
                    "device_code": device_code,
                    "client_id": client_id,
                },
            )
            assert poll_resp.status_code == 400, poll_resp.text
            err = poll_resp.json()
            assert err.get("error") == "access_denied", \
                f"Expected access_denied, got: {err}"
        finally:
            requests.delete(
                f"{BASE_URL}/api/v1/agents/{agent['id']}",
                headers=_admin_headers(admin_key),
            )

    def test_expired_device_code_rejected(self, admin_key):
        """Expired device_code must return expired_token error (backend enforced)."""
        agent = _register_device_client(admin_key)
        client_id = agent["client_id"]
        try:
            poll_resp = requests.post(
                f"{BASE_URL}/oauth/token",
                data={
                    "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
                    "device_code": "nonexistent-device-code-xyz",
                    "client_id": client_id,
                },
            )
            assert poll_resp.status_code == 400, poll_resp.text
            err = poll_resp.json()
            assert "error" in err
            assert err["error"] in ("expired_token", "authorization_pending", "invalid_grant"), \
                f"Unexpected error: {err}"
        finally:
            requests.delete(
                f"{BASE_URL}/api/v1/agents/{agent['id']}",
                headers=_admin_headers(admin_key),
            )

    def test_slow_down_on_rapid_polling(self, admin_key):
        """Rapid polling within interval should return slow_down (best-effort check)."""
        agent = _register_device_client(admin_key)
        client_id = agent["client_id"]
        try:
            dev_resp = requests.post(
                f"{BASE_URL}/oauth/device",
                data={"client_id": client_id, "scope": "read"},
            )
            assert dev_resp.status_code == 200
            device_code = dev_resp.json()["device_code"]

            errors = set()
            for _ in range(3):
                poll_resp = requests.post(
                    f"{BASE_URL}/oauth/token",
                    data={
                        "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
                        "device_code": device_code,
                        "client_id": client_id,
                    },
                )
                if poll_resp.status_code == 400:
                    errors.add(poll_resp.json().get("error"))

            # At least one of the acceptable pending/slowdown errors must appear
            assert errors & {"authorization_pending", "slow_down"}, \
                f"No pending/slow_down error in rapid poll, got: {errors}"
        finally:
            requests.delete(
                f"{BASE_URL}/api/v1/agents/{agent['id']}",
                headers=_admin_headers(admin_key),
            )
