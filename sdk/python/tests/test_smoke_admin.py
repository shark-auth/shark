"""Smoke tests — one happy-path call per untested namespace.

Verifies HTTP method + path construction only. No backend behavior.
Pattern: unittest.mock.patch("shark_auth._http.request") — same as test_users.py.
"""

from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest

BASE = "https://auth.example.com"
KEY = "sk_live_test"


def _resp(status: int, body: object) -> MagicMock:
    r = MagicMock()
    r.status_code = status
    r.json.return_value = body
    r.text = json.dumps(body)
    r.headers = {"Content-Type": "application/json"}
    return r


# ---------------------------------------------------------------------------
# api_keys
# ---------------------------------------------------------------------------

def test_api_keys_list():
    from shark_auth.api_keys import APIKeysClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"data": [{"id": "key_1", "name": "ci"}]})
        result = APIKeysClient(BASE, KEY).list()

    assert result[0]["id"] == "key_1"
    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/api-keys" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# apps
# ---------------------------------------------------------------------------

def test_apps_list():
    from shark_auth.apps import AppsClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"data": [{"id": "app_1", "name": "My App"}]})
        result = AppsClient(BASE, KEY).list()

    assert result[0]["id"] == "app_1"
    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/admin/apps" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# audit
# ---------------------------------------------------------------------------

def test_audit_list():
    from shark_auth.audit import AuditClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"data": [], "total": 0})
        AuditClient(BASE, KEY).list()

    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/audit-logs" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# auth
# ---------------------------------------------------------------------------

def test_auth_get_me():
    from shark_auth.auth import AuthClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"id": "usr_1", "email": "a@b.com"})
        result = AuthClient(BASE).get_me()

    assert result["email"] == "a@b.com"
    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/auth/me" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# consents
# ---------------------------------------------------------------------------

def test_consents_list():
    from shark_auth.consents import ConsentsClient
    import requests

    session = requests.Session()
    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, [{"id": "con_1", "app_id": "app_1"}])
        result = ConsentsClient(BASE, session=session).list()

    assert result[0]["id"] == "con_1"
    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/auth/consents" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# dcr
# ---------------------------------------------------------------------------

def test_dcr_get():
    from shark_auth.dcr import DCRClient

    client_id = "c_abc"
    rat = "rat_token"
    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"client_id": client_id, "client_name": "agent"})
        result = DCRClient(BASE).get(client_id, rat)

    assert result["client_id"] == client_id
    assert mock.call_args[0][1] == "GET"
    assert f"/oauth/register/{client_id}" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# magic_link
# ---------------------------------------------------------------------------

def test_magic_link_send():
    from shark_auth.magic_link import MagicLinkClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"sent": True})
        MagicLinkClient(BASE).send_magic_link("user@example.com")

    assert mock.call_args[0][1] == "POST"
    assert "/api/v1/auth/magic-link/send" in mock.call_args[0][2]
    assert mock.call_args[1]["json"]["email"] == "user@example.com"


# ---------------------------------------------------------------------------
# mfa
# ---------------------------------------------------------------------------

def test_mfa_enroll():
    from shark_auth.mfa import MFAClient
    import requests

    session = requests.Session()
    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"secret": "JBSWY3DPEHPK3PXP", "qr_url": "otpauth://…"})
        result = MFAClient(BASE, session=session).enroll()

    assert "secret" in result
    assert mock.call_args[0][1] == "POST"
    assert "/api/v1/auth/mfa/enroll" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# oauth
# ---------------------------------------------------------------------------

def test_oauth_introspect():
    from shark_auth.oauth import OAuthClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"active": True, "sub": "usr_1"})
        result = OAuthClient(BASE, KEY).introspect_token("tok_abc")

    assert result["active"] is True
    assert mock.call_args[0][1] == "POST"


# ---------------------------------------------------------------------------
# organizations
# ---------------------------------------------------------------------------

def test_organizations_list():
    from shark_auth.organizations import OrganizationsClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"data": [{"id": "org_1", "name": "Acme"}]})
        result = OrganizationsClient(BASE, KEY).list()

    assert result[0]["id"] == "org_1"
    assert mock.call_args[0][1] == "GET"
    assert "/admin/organizations" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# rbac
# ---------------------------------------------------------------------------

def test_rbac_list_roles():
    from shark_auth.rbac import RBACClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"data": [{"id": "role_1", "name": "admin"}]})
        result = RBACClient(BASE, KEY).list_roles()

    assert result[0]["id"] == "role_1"
    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/roles" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# sessions
# ---------------------------------------------------------------------------

def test_sessions_list():
    from shark_auth.sessions import SessionsClient
    import requests

    session = requests.Session()
    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, [{"id": "sess_1", "user_id": "usr_1"}])
        result = SessionsClient(BASE, session=session).list()

    assert result[0]["id"] == "sess_1"
    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/auth/sessions" in mock.call_args[0][2]


# ---------------------------------------------------------------------------
# webhooks
# ---------------------------------------------------------------------------

def test_webhooks_list():
    from shark_auth.webhooks import WebhooksClient

    with patch("shark_auth._http.request") as mock:
        mock.return_value = _resp(200, {"data": [{"id": "wh_1", "url": "https://hooks.example.com"}]})
        result = WebhooksClient(BASE, KEY).list()

    assert result[0]["id"] == "wh_1"
    assert mock.call_args[0][1] == "GET"
    assert "/api/v1/admin/webhooks" in mock.call_args[0][2]
