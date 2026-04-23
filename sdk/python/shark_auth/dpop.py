"""DPoP proof JWT generation per RFC 9449.

Holds an ECDSA P-256 keypair and emits DPoP proof JWTs bound to an HTTP
method, URL, and optionally an access token (via the ``ath`` claim).
"""

from __future__ import annotations

import base64
import hashlib
import json
import secrets
import time
from dataclasses import dataclass
from typing import Optional

import jwt
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import ec

from .errors import DPoPError


def _b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).rstrip(b"=").decode("ascii")


def _int_to_b64url(value: int, length: int) -> str:
    return _b64url(value.to_bytes(length, "big"))


@dataclass
class DPoPProver:
    """RFC 9449 DPoP prover.

    Typically constructed via :meth:`generate` (fresh P-256 key) or
    :meth:`from_pem` (load an existing PEM-encoded private key).
    """

    _private_key: ec.EllipticCurvePrivateKey
    _public_jwk: dict
    _thumbprint: str

    # ------------------------------------------------------------------
    # Constructors
    # ------------------------------------------------------------------
    @classmethod
    def generate(cls) -> "DPoPProver":
        """Generate a fresh ECDSA P-256 keypair."""
        key = ec.generate_private_key(ec.SECP256R1())
        return cls._from_private_key(key)

    @classmethod
    def from_pem(cls, pem: bytes, password: Optional[bytes] = None) -> "DPoPProver":
        """Load a PEM-encoded private key."""
        try:
            key = serialization.load_pem_private_key(pem, password=password)
        except Exception as exc:  # pragma: no cover - re-wrap
            raise DPoPError(f"failed to load private key: {exc}") from exc
        if not isinstance(key, ec.EllipticCurvePrivateKey) or not isinstance(
            key.curve, ec.SECP256R1
        ):
            raise DPoPError("private key must be ECDSA P-256 (secp256r1)")
        return cls._from_private_key(key)

    @classmethod
    def _from_private_key(cls, key: ec.EllipticCurvePrivateKey) -> "DPoPProver":
        numbers = key.public_key().public_numbers()
        x = _int_to_b64url(numbers.x, 32)
        y = _int_to_b64url(numbers.y, 32)
        jwk = {"kty": "EC", "crv": "P-256", "x": x, "y": y}
        thumbprint = _jwk_thumbprint(jwk)
        return cls(_private_key=key, _public_jwk=jwk, _thumbprint=thumbprint)

    # ------------------------------------------------------------------
    # Accessors
    # ------------------------------------------------------------------
    def private_key_pem(self) -> bytes:
        """Return the private key as unencrypted PEM (PKCS#8)."""
        return self._private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption(),
        )

    @property
    def public_jwk(self) -> dict:
        """Return the public key as a JWK (dict)."""
        return dict(self._public_jwk)

    @property
    def jkt(self) -> str:
        """Return the RFC 7638 JWK thumbprint (SHA-256, base64url)."""
        return self._thumbprint

    # ------------------------------------------------------------------
    # Proof emission
    # ------------------------------------------------------------------
    def make_proof(
        self,
        htm: str,
        htu: str,
        *,
        nonce: Optional[str] = None,
        access_token: Optional[str] = None,
        iat: Optional[int] = None,
        jti: Optional[str] = None,
    ) -> str:
        """Build and sign a DPoP proof JWT.

        Parameters
        ----------
        htm
            HTTP method (e.g. ``"POST"``). Case-insensitive; normalized to upper.
        htu
            Target URL (without query / fragment per RFC 9449 §4.2).
        nonce
            Optional DPoP nonce advertised by the server in a prior
            ``DPoP-Nonce`` response header.
        access_token
            If set, the proof binds to the token via the ``ath`` claim
            (``base64url(sha256(access_token))``).
        iat, jti
            Optional overrides for testing. In production, leave unset.
        """
        if not htm or not htu:
            raise DPoPError("htm and htu are required")

        payload = {
            "jti": jti or _b64url(secrets.token_bytes(16)),
            "htm": htm.upper(),
            "htu": htu,
            "iat": iat if iat is not None else int(time.time()),
        }
        if nonce is not None:
            payload["nonce"] = nonce
        if access_token is not None:
            digest = hashlib.sha256(access_token.encode("ascii")).digest()
            payload["ath"] = _b64url(digest)

        headers = {"typ": "dpop+jwt", "alg": "ES256", "jwk": self._public_jwk}
        pem = self.private_key_pem()
        try:
            return jwt.encode(payload, pem, algorithm="ES256", headers=headers)
        except Exception as exc:  # pragma: no cover
            raise DPoPError(f"failed to sign proof: {exc}") from exc


def _jwk_thumbprint(jwk: dict) -> str:
    """RFC 7638 JWK thumbprint for an EC JWK."""
    # Canonical form: lexicographic keys, no whitespace.
    canonical = {"crv": jwk["crv"], "kty": jwk["kty"], "x": jwk["x"], "y": jwk["y"]}
    serialized = json.dumps(canonical, separators=(",", ":"), sort_keys=True).encode(
        "utf-8"
    )
    return _b64url(hashlib.sha256(serialized).digest())
