# dpop.py

**Path:** `sdk/python/shark_auth/dpop.py`
**Module:** `shark_auth.dpop`
**LOC:** 163

## Purpose
RFC 9449 DPoP prover — holds an ECDSA P-256 keypair, exposes its public JWK + RFC 7638 thumbprint (`jkt`), and emits signed DPoP proof JWTs bound to a specific HTTP method, URL, and (optionally) access token.

## Public API
- `@dataclass class DPoPProver`
  - `DPoPProver.generate() -> DPoPProver` — fresh P-256 keypair
  - `DPoPProver.from_pem(pem: bytes, password=None) -> DPoPProver` — load existing PKCS#8 PEM
  - `.private_key_pem() -> bytes` — unencrypted PKCS#8 PEM export
  - `.public_jwk -> dict` — JWK form `{kty, crv, x, y}`
  - `.jkt -> str` — RFC 7638 SHA-256 thumbprint, base64url
  - `.make_proof(htm, htu, *, nonce=None, access_token=None, iat=None, jti=None) -> str` — signs ES256 JWT with `typ=dpop+jwt` header

## make_proof claims
- `jti` — random 16-byte b64url; overridable for tests
- `htm` — uppercased HTTP method
- `htu` — URL with query+fragment auto-stripped per RFC 9449 §4.2
- `iat` — current unix time; overridable
- `nonce` — when server advertises `DPoP-Nonce`
- `ath` — `b64url(sha256(access_token))` when token provided

## Internal dependencies
- `cryptography` (P-256 keygen, PEM IO)
- `PyJWT` (ES256 signing)
- `errors.DPoPError`

## Notes
- Curve enforcement: `from_pem` rejects anything that isn't `secp256r1`.
- Module-level helpers `_b64url`, `_int_to_b64url`, `_jwk_thumbprint` are private.
- Thumbprint canonicalization sorts keys lexicographically with no whitespace, per RFC 7638.
- This class is consumed by `AgentSession` (auto-signs every request) and `device_flow.DeviceFlow` / `tokens.exchange_token` (binds tokens at issuance).
