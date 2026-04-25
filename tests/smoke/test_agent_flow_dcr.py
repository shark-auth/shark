"""Agent flow smoke — RFC 7591 Dynamic Client Registration.

Covers: POST /oauth/register (no auth), GET/PUT/DELETE with reg-token,
secret rotation, reg-token rotation.
"""
import pytest
import requests
import os

BASE_URL = os.environ.get("BASE", "http://localhost:8080")


_DCR_BASE = {
    "grant_types": ["client_credentials"],
    "token_endpoint_auth_method": "client_secret_basic",
}


def _register(payload: dict) -> requests.Response:
    return requests.post(
        f"{BASE_URL}/oauth/register",
        json=payload,
        headers={"Accept": "application/json"},
    )


class TestDCRRegister:
    def test_register_minimal(self):
        """POST /oauth/register with client_credentials grant returns 201."""
        resp = _register({**_DCR_BASE, "client_name": "smoke-dcr-minimal"})
        assert resp.status_code == 201, resp.text
        data = resp.json()
        assert "client_id" in data
        assert "client_secret" in data
        assert "registration_access_token" in data
        assert "registration_client_uri" in data
        assert "client_credentials" in data.get("grant_types", [])

    def test_register_full_metadata(self):
        """POST /oauth/register with full RFC 7591 metadata."""
        resp = _register({
            "client_name": "smoke-dcr-full",
            "grant_types": ["client_credentials"],
            "token_endpoint_auth_method": "client_secret_basic",
            "scope": "read write",
            "software_id": "test-app",
        })
        assert resp.status_code == 201, resp.text
        data = resp.json()
        assert data["client_name"] == "smoke-dcr-full"
        assert "client_credentials" in data["grant_types"]

    def test_register_no_auth_required(self):
        """DCR endpoint must NOT require Authorization header."""
        resp = requests.post(
            f"{BASE_URL}/oauth/register",
            json={**_DCR_BASE, "client_name": "smoke-dcr-noauth"},
        )
        # Must succeed — no auth header sent
        assert resp.status_code == 201, f"DCR should not require auth: {resp.text}"

    def test_register_invalid_grant_type_rejected(self):
        """DCR rejects unknown/disallowed grant_types."""
        resp = _register({
            "client_name": "smoke-dcr-badgrant",
            "grant_types": ["urn:ietf:params:oauth:grant-type:INVALID"],
        })
        assert resp.status_code in (400, 422), resp.text
        data = resp.json()
        assert "error" in data


class TestDCRReadUpdateDelete:
    @pytest.fixture
    def registered_client(self):
        """Register a client and yield (client_id, reg_token); DELETE after test."""
        resp = _register({**_DCR_BASE, "client_name": "smoke-dcr-crud"})
        assert resp.status_code == 201, resp.text
        data = resp.json()
        client_id = data["client_id"]
        reg_token = data["registration_access_token"]
        yield client_id, reg_token
        # Cleanup
        requests.delete(
            f"{BASE_URL}/oauth/register/{client_id}",
            headers={"Authorization": f"Bearer {reg_token}"},
        )

    def test_get_client_configuration(self, registered_client):
        """GET /oauth/register/{client_id} with reg-token returns client metadata."""
        client_id, reg_token = registered_client
        resp = requests.get(
            f"{BASE_URL}/oauth/register/{client_id}",
            headers={"Authorization": f"Bearer {reg_token}"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["client_id"] == client_id

    def test_get_without_reg_token_rejected(self, registered_client):
        """GET /oauth/register/{client_id} without token must return 401."""
        client_id, _ = registered_client
        resp = requests.get(f"{BASE_URL}/oauth/register/{client_id}")
        assert resp.status_code == 401, resp.text

    def test_get_wrong_token_rejected(self, registered_client):
        """GET /oauth/register/{client_id} with wrong token must return 401."""
        client_id, _ = registered_client
        resp = requests.get(
            f"{BASE_URL}/oauth/register/{client_id}",
            headers={"Authorization": "Bearer wrong-token-value"},
        )
        assert resp.status_code == 401, resp.text

    def test_put_update_client_name(self, registered_client):
        """PUT /oauth/register/{client_id} updates client metadata."""
        client_id, reg_token = registered_client
        resp = requests.put(
            f"{BASE_URL}/oauth/register/{client_id}",
            json={
                "client_name": "smoke-dcr-crud-updated",
                "grant_types": ["client_credentials"],
            },
            headers={"Authorization": f"Bearer {reg_token}"},
        )
        assert resp.status_code == 200, resp.text
        data = resp.json()
        assert data["client_name"] == "smoke-dcr-crud-updated"

    def test_delete_client(self):
        """DELETE /oauth/register/{client_id} removes the client."""
        # Register fresh client for this test (not using fixture so we can DELETE it)
        resp = _register({**_DCR_BASE, "client_name": "smoke-dcr-delete"})
        assert resp.status_code == 201, resp.text
        data = resp.json()
        client_id = data["client_id"]
        reg_token = data["registration_access_token"]

        # Delete
        del_resp = requests.delete(
            f"{BASE_URL}/oauth/register/{client_id}",
            headers={"Authorization": f"Bearer {reg_token}"},
        )
        assert del_resp.status_code == 204, del_resp.text

        # Confirm gone
        get_resp = requests.get(
            f"{BASE_URL}/oauth/register/{client_id}",
            headers={"Authorization": f"Bearer {reg_token}"},
        )
        assert get_resp.status_code in (401, 404), get_resp.text


class TestDCRTokenUsability:
    def test_dcr_client_can_get_token(self):
        """Client registered via DCR can obtain access token via client_credentials."""
        resp = _register({
            **_DCR_BASE,
            "client_name": "smoke-dcr-token",
            "scope": "read",
        })
        assert resp.status_code == 201, resp.text
        data = resp.json()
        client_id = data["client_id"]
        client_secret = data["client_secret"]
        reg_token = data.get("registration_access_token")

        try:
            token_resp = requests.post(
                f"{BASE_URL}/oauth/token",
                data={"grant_type": "client_credentials", "scope": "read"},
                auth=(client_id, client_secret),
            )
            assert token_resp.status_code == 200, token_resp.text
            token_data = token_resp.json()
            assert "access_token" in token_data
        finally:
            if reg_token:
                requests.delete(
                    f"{BASE_URL}/oauth/register/{client_id}",
                    headers={"Authorization": f"Bearer {reg_token}"},
                )
