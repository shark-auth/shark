"""Tests for VaultClient."""

from __future__ import annotations

import pytest
from pytest_httpserver import HTTPServer
from werkzeug.wrappers import Response

from shark_auth import DPoPProver, VaultClient
from shark_auth.errors import VaultError


def _prover() -> DPoPProver:
    return DPoPProver.generate()


def test_fetch_token_happy_path(httpserver: HTTPServer):
    def handler(request):
        assert request.headers["Authorization"] == "DPoP agent_token_xyz"
        assert request.headers["DPoP"]
        return Response(
            '{"access_token":"ya29.fresh","token_type":"Bearer","expires_at":"2026-05-01T00:00:00Z","provider":"google"}',
            mimetype="application/json",
        )

    httpserver.expect_request(
        "/api/v1/vault/google/token", method="GET"
    ).respond_with_handler(handler)

    vault = VaultClient(auth_url=httpserver.url_for(""))
    tok = vault.fetch_token(
        provider="google", bearer_token="agent_token_xyz", prover=_prover()
    )
    assert tok.access_token == "ya29.fresh"
    assert tok.token_type == "Bearer"
    assert tok.expires_at == "2026-05-01T00:00:00Z"
    assert tok.provider == "google"


def test_fetch_token_defaults_token_type_and_provider(httpserver: HTTPServer):
    httpserver.expect_request(
        "/api/v1/vault/github/token", method="GET"
    ).respond_with_json(
        {"access_token": "t"}
    )
    vault = VaultClient(auth_url=httpserver.url_for(""))
    tok = vault.fetch_token(provider="github", bearer_token="at", prover=_prover())
    assert tok.access_token == "t"
    assert tok.token_type == "Bearer"
    assert tok.provider == "github"


def test_404_raises_not_found(httpserver: HTTPServer):
    httpserver.expect_request(
        "/api/v1/vault/missing/token", method="GET"
    ).respond_with_json({"error": "not_found"}, status=404)
    vault = VaultClient(auth_url=httpserver.url_for(""))
    with pytest.raises(VaultError) as exc:
        vault.fetch_token(provider="missing", bearer_token="at", prover=_prover())
    assert exc.value.status_code == 404


def test_401_raises_unauthorized(httpserver: HTTPServer):
    httpserver.expect_request(
        "/api/v1/vault/google/token", method="GET"
    ).respond_with_json({"error": "unauthorized"}, status=401)
    vault = VaultClient(auth_url=httpserver.url_for(""))
    with pytest.raises(VaultError) as exc:
        vault.fetch_token(provider="google", bearer_token="at", prover=_prover())
    assert exc.value.status_code == 401


def test_403_raises_forbidden(httpserver: HTTPServer):
    httpserver.expect_request(
        "/api/v1/vault/google/token", method="GET"
    ).respond_with_json({"error": "forbidden"}, status=403)
    vault = VaultClient(auth_url=httpserver.url_for(""))
    with pytest.raises(VaultError) as exc:
        vault.fetch_token(provider="google", bearer_token="at", prover=_prover())
    assert exc.value.status_code == 403


def test_get_fresh_token_is_removed():
    vault = VaultClient(auth_url="https://x.example", access_token="at")
    with pytest.raises(VaultError, match="get_fresh_token has been removed"):
        vault.get_fresh_token("conn_abc")
