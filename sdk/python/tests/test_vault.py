"""Tests for VaultClient."""

from __future__ import annotations

import pytest
from pytest_httpserver import HTTPServer

from shark_auth import VaultClient
from shark_auth.errors import VaultError


def test_get_fresh_token_happy_path(httpserver: HTTPServer):
    httpserver.expect_request(
        "/admin/vault/connections/conn_abc/token",
        method="GET",
        headers={"Authorization": "Bearer agent_token_xyz"},
    ).respond_with_json(
        {
            "access_token": "ya29.fresh",
            "expires_at": 1999999999,
            "provider": "google",
            "scopes": ["gmail.readonly", "calendar.events"],
        }
    )

    vault = VaultClient(auth_url=httpserver.url_for(""), access_token="agent_token_xyz")
    tok = vault.get_fresh_token("conn_abc")
    assert tok.access_token == "ya29.fresh"
    assert tok.expires_at == 1999999999
    assert tok.provider == "google"
    assert tok.scopes == ["gmail.readonly", "calendar.events"]


def test_scopes_as_space_delimited_string(httpserver: HTTPServer):
    httpserver.expect_request(
        "/admin/vault/connections/conn_x/token", method="GET"
    ).respond_with_json(
        {"access_token": "t", "scope": "read write", "provider": "github"}
    )
    vault = VaultClient(auth_url=httpserver.url_for(""), access_token="at")
    tok = vault.get_fresh_token("conn_x")
    assert tok.scopes == ["read", "write"]


def test_404_raises_not_found(httpserver: HTTPServer):
    httpserver.expect_request(
        "/admin/vault/connections/missing/token", method="GET"
    ).respond_with_json({"error": "not_found"}, status=404)
    vault = VaultClient(auth_url=httpserver.url_for(""), access_token="at")
    with pytest.raises(VaultError) as exc:
        vault.get_fresh_token("missing")
    assert exc.value.status_code == 404


def test_401_raises_unauthorized(httpserver: HTTPServer):
    httpserver.expect_request(
        "/admin/vault/connections/conn/token", method="GET"
    ).respond_with_json({"error": "unauthorized"}, status=401)
    vault = VaultClient(auth_url=httpserver.url_for(""), access_token="at")
    with pytest.raises(VaultError) as exc:
        vault.get_fresh_token("conn")
    assert exc.value.status_code == 401


def test_403_raises_forbidden(httpserver: HTTPServer):
    httpserver.expect_request(
        "/admin/vault/connections/conn/token", method="GET"
    ).respond_with_json({"error": "forbidden"}, status=403)
    vault = VaultClient(auth_url=httpserver.url_for(""), access_token="at")
    with pytest.raises(VaultError) as exc:
        vault.get_fresh_token("conn")
    assert exc.value.status_code == 403


def test_empty_connection_id_rejected():
    vault = VaultClient(auth_url="https://x.example", access_token="at")
    with pytest.raises(VaultError):
        vault.get_fresh_token("")
