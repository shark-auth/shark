"""Tests for DPoPProver (RFC 9449)."""

from __future__ import annotations

import base64
import hashlib
import json

import jwt
import pytest

from shark_auth import DPoPProver
from shark_auth.errors import DPoPError


def _b64url_decode(s: str) -> bytes:
    pad = "=" * (-len(s) % 4)
    return base64.urlsafe_b64decode(s + pad)


def test_generate_emits_valid_header_and_payload():
    prover = DPoPProver.generate()
    proof = prover.make_proof("POST", "https://auth.example/oauth/token")

    header = jwt.get_unverified_header(proof)
    assert header["typ"] == "dpop+jwt"
    assert header["alg"] == "ES256"
    assert header["jwk"]["kty"] == "EC"
    assert header["jwk"]["crv"] == "P-256"
    assert set(header["jwk"]) == {"kty", "crv", "x", "y"}  # no private material

    # Decode without verification to inspect claims
    payload = jwt.decode(proof, options={"verify_signature": False})
    assert payload["htm"] == "POST"
    assert payload["htu"] == "https://auth.example/oauth/token"
    assert "jti" in payload and len(payload["jti"]) >= 16
    assert isinstance(payload["iat"], int)
    assert "ath" not in payload
    assert "nonce" not in payload


def test_signature_verifies_with_embedded_jwk():
    prover = DPoPProver.generate()
    proof = prover.make_proof("GET", "https://api.example/data")
    header = jwt.get_unverified_header(proof)
    # Re-verify using the embedded JWK
    public_key = jwt.algorithms.ECAlgorithm.from_jwk(json.dumps(header["jwk"]))
    decoded = jwt.decode(proof, key=public_key, algorithms=["ES256"])
    assert decoded["htm"] == "GET"


def test_ath_claim_set_when_access_token_provided():
    prover = DPoPProver.generate()
    token = "abc.def.ghi"
    proof = prover.make_proof("GET", "https://api.example/x", access_token=token)
    payload = jwt.decode(proof, options={"verify_signature": False})
    expected = base64.urlsafe_b64encode(
        hashlib.sha256(token.encode("ascii")).digest()
    ).rstrip(b"=").decode("ascii")
    assert payload["ath"] == expected


def test_nonce_claim_set_when_provided():
    prover = DPoPProver.generate()
    proof = prover.make_proof("POST", "https://auth.example/t", nonce="server-nonce-1")
    payload = jwt.decode(proof, options={"verify_signature": False})
    assert payload["nonce"] == "server-nonce-1"


def test_jti_uniqueness_across_many_proofs():
    prover = DPoPProver.generate()
    jtis = set()
    for _ in range(100):
        proof = prover.make_proof("POST", "https://x.example/t")
        payload = jwt.decode(proof, options={"verify_signature": False})
        jtis.add(payload["jti"])
    assert len(jtis) == 100


def test_thumbprint_stable_across_proofs():
    prover = DPoPProver.generate()
    t1 = prover.jkt
    prover.make_proof("POST", "https://x.example/t")
    t2 = prover.jkt
    assert t1 == t2
    # Thumbprint must be 32 bytes (SHA-256) base64url-encoded = 43 chars unpadded
    raw = _b64url_decode(t1)
    assert len(raw) == 32


def test_from_pem_roundtrip_preserves_thumbprint():
    p1 = DPoPProver.generate()
    pem = p1.private_key_pem()
    p2 = DPoPProver.from_pem(pem)
    assert p1.jkt == p2.jkt
    # Both can produce verifiable proofs
    proof = p2.make_proof("POST", "https://x.example/t")
    header = jwt.get_unverified_header(proof)
    public_key = jwt.algorithms.ECAlgorithm.from_jwk(json.dumps(header["jwk"]))
    jwt.decode(proof, key=public_key, algorithms=["ES256"])


def test_from_pem_rejects_non_p256():
    from cryptography.hazmat.primitives import serialization
    from cryptography.hazmat.primitives.asymmetric import ec

    key = ec.generate_private_key(ec.SECP384R1())
    pem = key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.NoEncryption(),
    )
    with pytest.raises(DPoPError):
        DPoPProver.from_pem(pem)


def test_htm_and_htu_required():
    prover = DPoPProver.generate()
    with pytest.raises(DPoPError):
        prover.make_proof("", "https://x.example/t")
    with pytest.raises(DPoPError):
        prover.make_proof("POST", "")


def test_htm_normalized_to_upper():
    prover = DPoPProver.generate()
    proof = prover.make_proof("post", "https://x.example/t")
    payload = jwt.decode(proof, options={"verify_signature": False})
    assert payload["htm"] == "POST"
