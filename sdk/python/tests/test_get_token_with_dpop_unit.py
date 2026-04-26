"""Unit tests for OAuthClient.get_token_with_dpop().

Uses ``responses`` to mock the token endpoint so no live server is needed.
"""

from __future__ import annotations

import json

import jwt
import pytest
import responses as resp_lib

from shark_auth.dpop import DPoPProver
from shark_auth.errors import OAuthError
from shark_auth.oauth import OAuthClient, Token

BASE_URL = "https://auth.example.com"
TOKEN_ENDPOINT = f"{BASE_URL}/oauth/token"

# Minimal happy-path server response.
MOCK_TOKEN_RESPONSE = {
    "access_token": "eyJhbGciOiJFUzI1NiJ9.stub",
    "token_type": "DPoP",
    "expires_in": 3600,
    "scope": "read write",
    "cnf": {"jkt": "__placeholder__"},  # filled in per-test with real thumbprint
}


@pytest.fixture()
def prover() -> DPoPProver:
    return DPoPProver.generate()


@pytest.fixture()
def client() -> OAuthClient:
    return OAuthClient(base_url=BASE_URL)


# ---------------------------------------------------------------------------
# Happy path — client_credentials
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_happy_path_client_credentials(client: OAuthClient, prover: DPoPProver) -> None:
    """200 response yields a populated Token with cnf_jkt matching the prover."""
    mock_body = {**MOCK_TOKEN_RESPONSE, "cnf": {"jkt": prover.jkt}}

    resp_lib.add(
        resp_lib.POST,
        TOKEN_ENDPOINT,
        json=mock_body,
        status=200,
    )

    token = client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id="my-agent",
        client_secret="s3cr3t",
        scope="read write",
    )

    assert isinstance(token, Token)
    assert token.access_token == mock_body["access_token"]
    assert token.token_type == "DPoP"
    assert token.expires_in == 3600
    assert token.scope == "read write"
    assert token.cnf_jkt == prover.jkt
    assert token.raw == mock_body


@resp_lib.activate
def test_request_body_fields(client: OAuthClient, prover: DPoPProver) -> None:
    """Verify the form body contains grant_type, client_id, and scope."""
    captured: dict = {}

    def capture(request):  # type: ignore[no-untyped-def]
        # requests sends form data as bytes; decode and parse.
        from urllib.parse import parse_qs
        captured.update(
            {k: v[0] for k, v in parse_qs(request.body).items()}
        )
        mock_body = {**MOCK_TOKEN_RESPONSE, "cnf": {"jkt": prover.jkt}}
        return (200, {}, json.dumps(mock_body))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id="agent-123",
        client_secret="secret",
        scope="agents:read",
    )

    assert captured["grant_type"] == "client_credentials"
    assert captured["client_id"] == "agent-123"
    assert captured["client_secret"] == "secret"
    assert captured["scope"] == "agents:read"


@resp_lib.activate
def test_dpop_header_is_valid_jwt(client: OAuthClient, prover: DPoPProver) -> None:
    """DPoP header must be a parseable JWT with typ=dpop+jwt and alg=ES256."""
    captured_headers: dict = {}

    def capture(request):  # type: ignore[no-untyped-def]
        captured_headers.update(dict(request.headers))
        mock_body = {**MOCK_TOKEN_RESPONSE, "cnf": {"jkt": prover.jkt}}
        return (200, {}, json.dumps(mock_body))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id="my-agent",
    )

    assert "DPoP" in captured_headers, "DPoP header missing from request"
    proof = captured_headers["DPoP"]
    assert proof, "DPoP header is empty"

    # Parse without verification — we just check structure.
    header = jwt.get_unverified_header(proof)
    assert header["typ"] == "dpop+jwt"
    assert header["alg"] == "ES256"
    assert "jwk" in header


@resp_lib.activate
def test_no_scope_omits_scope_field(client: OAuthClient, prover: DPoPProver) -> None:
    """When scope=None, the form body must NOT include a scope key."""
    captured: dict = {}

    def capture(request):  # type: ignore[no-untyped-def]
        from urllib.parse import parse_qs
        captured.update(
            {k: v[0] for k, v in parse_qs(request.body).items()}
        )
        mock_body = {**MOCK_TOKEN_RESPONSE, "cnf": {"jkt": prover.jkt}}
        return (200, {}, json.dumps(mock_body))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id="my-agent",
        # scope deliberately omitted
    )

    assert "scope" not in captured


@resp_lib.activate
def test_extra_kwargs_forwarded(client: OAuthClient, prover: DPoPProver) -> None:
    """Extra kwargs (e.g. refresh_token) must appear in the form body."""
    captured: dict = {}

    def capture(request):  # type: ignore[no-untyped-def]
        from urllib.parse import parse_qs
        captured.update(
            {k: v[0] for k, v in parse_qs(request.body).items()}
        )
        mock_body = {**MOCK_TOKEN_RESPONSE, "cnf": {"jkt": prover.jkt}}
        return (200, {}, json.dumps(mock_body))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    client.get_token_with_dpop(
        grant_type="refresh_token",
        dpop_prover=prover,
        client_id="my-agent",
        refresh_token="rt_abc123",
    )

    assert captured["refresh_token"] == "rt_abc123"


# ---------------------------------------------------------------------------
# Error path — 4xx raises OAuthError
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_invalid_client_raises_oauth_error(client: OAuthClient, prover: DPoPProver) -> None:
    """401 invalid_client response raises OAuthError with correct fields."""
    resp_lib.add(
        resp_lib.POST,
        TOKEN_ENDPOINT,
        json={"error": "invalid_client", "error_description": "bad secret"},
        status=401,
    )

    with pytest.raises(OAuthError) as exc_info:
        client.get_token_with_dpop(
            grant_type="client_credentials",
            dpop_prover=prover,
            client_id="agent-x",
            client_secret="wrong",
        )

    err = exc_info.value
    assert err.error == "invalid_client"
    assert err.error_description == "bad secret"
    assert err.status_code == 401


@resp_lib.activate
def test_server_error_raises_oauth_error(client: OAuthClient, prover: DPoPProver) -> None:
    """500 response raises OAuthError."""
    resp_lib.add(
        resp_lib.POST,
        TOKEN_ENDPOINT,
        json={"error": "server_error", "error_description": "internal error"},
        status=500,
    )

    with pytest.raises(OAuthError) as exc_info:
        client.get_token_with_dpop(
            grant_type="client_credentials",
            dpop_prover=prover,
            client_id="agent-x",
        )

    assert exc_info.value.status_code == 500


@resp_lib.activate
def test_cnf_jkt_absent_when_server_omits_it(client: OAuthClient, prover: DPoPProver) -> None:
    """When server omits cnf.jkt, Token.cnf_jkt is None (not an error)."""
    mock_body = {
        "access_token": "tok",
        "token_type": "DPoP",
        "expires_in": 900,
    }
    resp_lib.add(resp_lib.POST, TOKEN_ENDPOINT, json=mock_body, status=200)

    token = client.get_token_with_dpop(
        grant_type="client_credentials",
        dpop_prover=prover,
        client_id="my-agent",
    )

    assert token.cnf_jkt is None
    assert token.access_token == "tok"
