"""Tests for decode_agent_token + JWKS cache."""

from __future__ import annotations

import json
import time

import jwt
import pytest
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from pytest_httpserver import HTTPServer

from shark_auth import decode_agent_token
from shark_auth import tokens as tokens_mod
from shark_auth.errors import TokenError


ISS = "https://auth.example"
AUD = "https://api.my-app.example"


def _gen_rsa_keypair():
    return rsa.generate_private_key(public_exponent=65537, key_size=2048)


def _jwk_for_rsa_public(key, kid: str) -> dict:
    numbers = key.public_key().public_numbers()
    import base64

    def b64(i: int) -> str:
        raw = i.to_bytes((i.bit_length() + 7) // 8, "big")
        return base64.urlsafe_b64encode(raw).rstrip(b"=").decode("ascii")

    return {
        "kty": "RSA",
        "use": "sig",
        "alg": "RS256",
        "kid": kid,
        "n": b64(numbers.n),
        "e": b64(numbers.e),
    }


def _sign(claims: dict, key, kid: str) -> str:
    pem = key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )
    return jwt.encode(claims, pem, algorithm="RS256", headers={"kid": kid})


def _base_claims(exp_in: int = 300) -> dict:
    now = int(time.time())
    return {
        "iss": ISS,
        "aud": AUD,
        "sub": "agent_abc",
        "iat": now,
        "exp": now + exp_in,
        "scope": "vault:read agent:exec",
        "agent_id": "agent_abc",
        "act": {"sub": "user_42"},
        "cnf": {"jkt": "abc123thumbprint"},
        "authorization_details": [
            {"type": "vault_access", "connection_id": "conn_x"}
        ],
    }


def test_decode_happy_path_preserves_extended_claims(httpserver: HTTPServer):
    key = _gen_rsa_keypair()
    kid = "k1"
    jwks = {"keys": [_jwk_for_rsa_public(key, kid)]}
    httpserver.expect_request("/.well-known/jwks.json").respond_with_json(jwks)

    token = _sign(_base_claims(), key, kid)
    claims = decode_agent_token(
        token=token,
        jwks_url=httpserver.url_for("/.well-known/jwks.json"),
        expected_issuer=ISS,
        expected_audience=AUD,
    )
    assert claims.sub == "agent_abc"
    assert claims.agent_id == "agent_abc"
    assert claims.scope == "vault:read agent:exec"
    assert claims.act == {"sub": "user_42"}
    assert claims.cnf == {"jkt": "abc123thumbprint"}
    assert claims.jkt == "abc123thumbprint"
    assert claims.authorization_details == [
        {"type": "vault_access", "connection_id": "conn_x"}
    ]


def test_wrong_issuer_raises(httpserver: HTTPServer):
    key = _gen_rsa_keypair()
    jwks = {"keys": [_jwk_for_rsa_public(key, "k1")]}
    httpserver.expect_request("/.well-known/jwks.json").respond_with_json(jwks)
    token = _sign(_base_claims(), key, "k1")
    with pytest.raises(TokenError, match="issuer"):
        decode_agent_token(
            token=token,
            jwks_url=httpserver.url_for("/.well-known/jwks.json"),
            expected_issuer="https://evil.example",
            expected_audience=AUD,
        )


def test_wrong_audience_raises(httpserver: HTTPServer):
    key = _gen_rsa_keypair()
    jwks = {"keys": [_jwk_for_rsa_public(key, "k1")]}
    httpserver.expect_request("/.well-known/jwks.json").respond_with_json(jwks)
    token = _sign(_base_claims(), key, "k1")
    with pytest.raises(TokenError, match="audience"):
        decode_agent_token(
            token=token,
            jwks_url=httpserver.url_for("/.well-known/jwks.json"),
            expected_issuer=ISS,
            expected_audience="https://api.other.example",
        )


def test_expired_raises(httpserver: HTTPServer):
    key = _gen_rsa_keypair()
    jwks = {"keys": [_jwk_for_rsa_public(key, "k1")]}
    httpserver.expect_request("/.well-known/jwks.json").respond_with_json(jwks)
    claims = _base_claims(exp_in=-30)
    token = _sign(claims, key, "k1")
    with pytest.raises(TokenError, match="expired"):
        decode_agent_token(
            token=token,
            jwks_url=httpserver.url_for("/.well-known/jwks.json"),
            expected_issuer=ISS,
            expected_audience=AUD,
        )


def test_tampered_signature_raises(httpserver: HTTPServer):
    key = _gen_rsa_keypair()
    jwks = {"keys": [_jwk_for_rsa_public(key, "k1")]}
    httpserver.expect_request("/.well-known/jwks.json").respond_with_json(jwks)
    token = _sign(_base_claims(), key, "k1")
    # Flip bits in the decoded signature, then re-encode. Avoids the edge case
    # where flipping a single base64 char toggles only padding bits.
    import base64

    head, payload, sig = token.split(".")
    pad = "=" * (-len(sig) % 4)
    raw = bytearray(base64.urlsafe_b64decode(sig + pad))
    for i in range(min(8, len(raw))):
        raw[i] ^= 0xFF
    tampered_sig = base64.urlsafe_b64encode(bytes(raw)).rstrip(b"=").decode("ascii")
    tampered = f"{head}.{payload}.{tampered_sig}"
    with pytest.raises(TokenError):
        decode_agent_token(
            token=tampered,
            jwks_url=httpserver.url_for("/.well-known/jwks.json"),
            expected_issuer=ISS,
            expected_audience=AUD,
        )


def test_kid_miss_triggers_refresh_and_then_succeeds(httpserver: HTTPServer):
    key_old = _gen_rsa_keypair()
    key_new = _gen_rsa_keypair()

    # Prime cache with only the old key
    old_jwks = {"keys": [_jwk_for_rsa_public(key_old, "old")]}
    new_jwks = {
        "keys": [
            _jwk_for_rsa_public(key_old, "old"),
            _jwk_for_rsa_public(key_new, "new"),
        ]
    }

    # First call (initial fetch) returns old_jwks, second call (refresh) returns new_jwks.
    import threading

    call_count = {"n": 0}
    lock = threading.Lock()

    def handler(request):
        from werkzeug.wrappers import Response

        with lock:
            call_count["n"] += 1
            n = call_count["n"]
        body = old_jwks if n == 1 else new_jwks
        return Response(json.dumps(body), status=200, mimetype="application/json")

    httpserver.expect_request("/.well-known/jwks.json").respond_with_handler(handler)

    url = httpserver.url_for("/.well-known/jwks.json")

    # Token signed with the NEW kid — not yet in the cached JWKS.
    # First decode should warm the cache (miss), refresh, find "new", succeed.
    token = _sign(_base_claims(), key_new, "new")
    claims = decode_agent_token(
        token=token,
        jwks_url=url,
        expected_issuer=ISS,
        expected_audience=AUD,
    )
    assert claims.sub == "agent_abc"
    assert call_count["n"] == 2  # initial fetch + refresh


def test_alg_none_rejected(httpserver: HTTPServer):
    key = _gen_rsa_keypair()
    jwks = {"keys": [_jwk_for_rsa_public(key, "k1")]}
    httpserver.expect_request("/.well-known/jwks.json").respond_with_json(jwks)
    # Manually craft an alg=none token
    import base64

    def b64(b: bytes) -> str:
        return base64.urlsafe_b64encode(b).rstrip(b"=").decode("ascii")

    header = b64(json.dumps({"alg": "none", "kid": "k1"}).encode())
    payload = b64(json.dumps(_base_claims()).encode())
    tok = f"{header}.{payload}."
    with pytest.raises(TokenError):
        decode_agent_token(
            token=tok,
            jwks_url=httpserver.url_for("/.well-known/jwks.json"),
            expected_issuer=ISS,
            expected_audience=AUD,
        )


def test_missing_kid_rejected(httpserver: HTTPServer):
    key = _gen_rsa_keypair()
    pem = key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )
    token = jwt.encode(_base_claims(), pem, algorithm="RS256")  # no kid
    with pytest.raises(TokenError, match="kid"):
        decode_agent_token(
            token=token,
            jwks_url="https://unused.example/jwks",
            expected_issuer=ISS,
            expected_audience=AUD,
        )
