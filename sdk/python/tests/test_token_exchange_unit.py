"""Unit tests for OAuthClient.token_exchange() — RFC 8693.

Uses ``responses`` to mock the token endpoint so no live server is needed.
Mirrors the style of test_get_token_with_dpop_unit.py.
"""

from __future__ import annotations

import json
from urllib.parse import parse_qs

import jwt
import pytest
import responses as resp_lib

from shark_auth.dpop import DPoPProver
from shark_auth.errors import OAuthError
from shark_auth.oauth import OAuthClient, Token

BASE_URL = "https://auth.example.com"
TOKEN_ENDPOINT = f"{BASE_URL}/oauth/token"

GRANT_TYPE_EXCHANGE = "urn:ietf:params:oauth:grant-type:token-exchange"
TOKEN_TYPE_ACCESS = "urn:ietf:params:oauth:token-type:access_token"

SUBJECT_TOKEN = "eyJhbGciOiJFUzI1NiJ9.parent-stub"

MOCK_EXCHANGE_RESPONSE = {
    "access_token": "eyJhbGciOiJFUzI1NiJ9.child-stub",
    "token_type": "DPoP",
    "expires_in": 1800,
    "scope": "mcp:read",
}


@pytest.fixture()
def prover() -> DPoPProver:
    return DPoPProver.generate()


@pytest.fixture()
def client() -> OAuthClient:
    return OAuthClient(base_url=BASE_URL)


# ---------------------------------------------------------------------------
# 1. Happy path — verify form body + DPoP header
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_happy_path_form_body_and_dpop(client: OAuthClient, prover: DPoPProver) -> None:
    """200 response: correct grant_type + subject fields; DPoP header present."""
    captured: dict = {}
    captured_headers: dict = {}

    mock_body = {**MOCK_EXCHANGE_RESPONSE, "cnf": {"jkt": prover.jkt}}

    def capture(request):  # type: ignore[no-untyped-def]
        captured.update({k: v[0] for k, v in parse_qs(request.body).items()})
        captured_headers.update(dict(request.headers))
        return (200, {}, json.dumps(mock_body))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    token = client.token_exchange(
        subject_token=SUBJECT_TOKEN,
        client_id="agent-a",
        client_secret="s3cr3t",
        dpop_prover=prover,
        scope="mcp:read",
    )

    # Response parsed correctly.
    assert isinstance(token, Token)
    assert token.access_token == mock_body["access_token"]
    assert token.token_type == "DPoP"
    assert token.expires_in == 1800
    assert token.scope == "mcp:read"
    assert token.cnf_jkt == prover.jkt

    # Required RFC 8693 form fields.
    assert captured["grant_type"] == GRANT_TYPE_EXCHANGE
    assert captured["client_id"] == "agent-a"
    assert captured["client_secret"] == "s3cr3t"
    assert captured["subject_token"] == SUBJECT_TOKEN
    assert captured["subject_token_type"] == TOKEN_TYPE_ACCESS
    assert captured["requested_token_type"] == TOKEN_TYPE_ACCESS
    assert captured["scope"] == "mcp:read"

    # DPoP header is a valid JWT.
    assert "DPoP" in captured_headers
    header = jwt.get_unverified_header(captured_headers["DPoP"])
    assert header["typ"] == "dpop+jwt"
    assert header["alg"] == "ES256"
    assert "jwk" in header


# ---------------------------------------------------------------------------
# 2. With audience — body must include audience field
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_with_audience(client: OAuthClient, prover: DPoPProver) -> None:
    """audience parameter appears in form body when provided."""
    captured: dict = {}

    def capture(request):  # type: ignore[no-untyped-def]
        captured.update({k: v[0] for k, v in parse_qs(request.body).items()})
        return (200, {}, json.dumps(MOCK_EXCHANGE_RESPONSE))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    client.token_exchange(
        subject_token=SUBJECT_TOKEN,
        client_id="agent-a",
        client_secret="s3cr3t",
        dpop_prover=prover,
        audience="https://mcp.example.com",
    )

    assert captured["audience"] == "https://mcp.example.com"
    # scope omitted when not given
    assert "scope" not in captured


# ---------------------------------------------------------------------------
# 3. With client_id + client_secret — body includes auth fields
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_with_client_auth(client: OAuthClient, prover: DPoPProver) -> None:
    """client_id and client_secret both appear in the form body."""
    captured: dict = {}

    def capture(request):  # type: ignore[no-untyped-def]
        captured.update({k: v[0] for k, v in parse_qs(request.body).items()})
        return (200, {}, json.dumps(MOCK_EXCHANGE_RESPONSE))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    client.token_exchange(
        subject_token=SUBJECT_TOKEN,
        client_id="agent-a",
        client_secret="s3cr3t",
        dpop_prover=prover,
    )

    assert captured["client_id"] == "agent-a"
    assert captured["client_secret"] == "s3cr3t"


# ---------------------------------------------------------------------------
# 4. Missing scope — form body must NOT contain scope key
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_missing_scope_omits_scope_field(client: OAuthClient, prover: DPoPProver) -> None:
    """When scope=None (default), the form body must NOT include a scope key."""
    captured: dict = {}

    def capture(request):  # type: ignore[no-untyped-def]
        captured.update({k: v[0] for k, v in parse_qs(request.body).items()})
        return (200, {}, json.dumps(MOCK_EXCHANGE_RESPONSE))

    resp_lib.add_callback(resp_lib.POST, TOKEN_ENDPOINT, callback=capture)

    client.token_exchange(
        subject_token=SUBJECT_TOKEN,
        client_id="agent-a",
        client_secret="s3cr3t",
        dpop_prover=prover,
        # scope deliberately omitted
    )

    assert "scope" not in captured


# ---------------------------------------------------------------------------
# 5. 400 invalid_scope raises OAuthError
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_invalid_scope_raises_oauth_error(client: OAuthClient, prover: DPoPProver) -> None:
    """400 invalid_scope raises OAuthError with correct fields."""
    resp_lib.add(
        resp_lib.POST,
        TOKEN_ENDPOINT,
        json={
            "error": "invalid_scope",
            "error_description": "requested scope exceeds the parent grant",
        },
        status=400,
    )

    with pytest.raises(OAuthError) as exc_info:
        client.token_exchange(
            subject_token=SUBJECT_TOKEN,
            client_id="agent-a",
            client_secret="s3cr3t",
            dpop_prover=prover,
            scope="mcp:admin",
        )

    err = exc_info.value
    assert err.error == "invalid_scope"
    assert err.status_code == 400


# ---------------------------------------------------------------------------
# 6. 401 invalid_token (revoked subject) raises OAuthError
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_revoked_subject_token_raises_oauth_error(client: OAuthClient, prover: DPoPProver) -> None:
    """401 invalid_token response raises OAuthError."""
    resp_lib.add(
        resp_lib.POST,
        TOKEN_ENDPOINT,
        json={
            "error": "invalid_token",
            "error_description": "subject_token has been revoked",
        },
        status=401,
    )

    with pytest.raises(OAuthError) as exc_info:
        client.token_exchange(
            subject_token="revoked-token",
            client_id="agent-a",
            client_secret="s3cr3t",
            dpop_prover=prover,
        )

    err = exc_info.value
    assert err.error == "invalid_token"
    assert err.status_code == 401


# ---------------------------------------------------------------------------
# 7. cnf_jkt populated from server response; None when absent
# ---------------------------------------------------------------------------

@resp_lib.activate
def test_cnf_jkt_populated_when_present(client: OAuthClient, prover: DPoPProver) -> None:
    """Token.cnf_jkt is set from cnf.jkt in the server response."""
    mock_body = {**MOCK_EXCHANGE_RESPONSE, "cnf": {"jkt": prover.jkt}}
    resp_lib.add(resp_lib.POST, TOKEN_ENDPOINT, json=mock_body, status=200)

    token = client.token_exchange(
        subject_token=SUBJECT_TOKEN,
        client_id="agent-a",
        client_secret="s3cr3t",
        dpop_prover=prover,
    )

    assert token.cnf_jkt == prover.jkt


@resp_lib.activate
def test_cnf_jkt_none_when_absent(client: OAuthClient, prover: DPoPProver) -> None:
    """Token.cnf_jkt is None when server response omits cnf.jkt."""
    resp_lib.add(resp_lib.POST, TOKEN_ENDPOINT, json=MOCK_EXCHANGE_RESPONSE, status=200)

    token = client.token_exchange(
        subject_token=SUBJECT_TOKEN,
        client_id="agent-a",
        client_secret="s3cr3t",
        dpop_prover=prover,
    )

    assert token.cnf_jkt is None
