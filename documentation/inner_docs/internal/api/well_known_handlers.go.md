# well_known_handlers.go

**Path:** `internal/api/well_known_handlers.go`
**Package:** `api`
**LOC:** 116
**Tests:** likely integration-tested via JWT manager tests

## Purpose
RFC 7517 JWKS publication endpoint at `/.well-known/jwks.json`. Builds JWK entries from PEM-encoded RSA + ECDSA public keys stored in the DB, including recently-retired keys (within 2x access-token TTL window) so in-flight tokens stay verifiable across rotations.

## Handlers exposed
- `(*Server).HandleJWKS` (line 73) — GET `/.well-known/jwks.json`. Reads keys via `s.Store.ListJWKSCandidates(ctx, false, retiredCutoff)` where `retiredCutoff = now - 2 * AccessTokenTTL`. Sets `Cache-Control: public, max-age=300`. Skips malformed PEMs rather than returning a broken JWKS.

## Key types
None — JWK entries are emitted as `map[string]interface{}` matching RFC 7517.

## Helpers
- `parsePEMPublicKey` (line 18) — PKIX-encoded RSA public key.
- `parseECPublicKeyPEM` (line 35) — PKIX-encoded ECDSA public key.
- `jwkFromPublicKey` (line 52) — builds RSA JWK (`kty=RSA, use=sig, alg=RS256, kid, n, e`).

## Algorithm support
- `ES256` → delegates to `jwtpkg.ES256PublicJWK` (defined in `internal/auth/jwt`).
- `RS256` (and any unrecognised) → falls through to RSA path.

## Imports of note
- `crypto/ecdsa`, `crypto/rsa`, `crypto/x509`, `encoding/pem`, `encoding/base64`
- `internal/auth/jwt` — `ES256PublicJWK`
- `s.Store.ListJWKSCandidates` (storage)

## Wired by
- Mounted at `/.well-known/jwks.json` in the router (oauth/jwt route group).

## Notes
- `Cache-Control: public, max-age=300` — 5-minute cache; clients that respect it won't re-fetch on every token verify.
- Retired-key window = 2× AccessTokenTTL keeps JWTs minted just before a rotation verifiable until they expire.
- Malformed PEM rows are silently skipped — one bad row never breaks the whole JWKS.
