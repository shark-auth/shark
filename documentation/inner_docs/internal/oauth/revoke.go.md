# revoke.go

**Path:** `internal/oauth/revoke.go`
**Package:** `oauth`
**LOC:** 122
**Tests:** covered by `handlers_test.go` + `store_test.go`

## Purpose
RFC 7009 Token Revocation endpoint with token-family reuse detection.

## RFCs implemented
- RFC 7009 OAuth 2.0 Token Revocation

## Key types / functions
- `HandleRevoke` (func, line 16) — authenticates caller (delegated to `authenticateClient` in introspect.go), locates token via shared `findTokenInDB`, enforces "client may only revoke own tokens" (admin bypass), revokes, and cascades:
  - refresh tokens (line 78) → `RevokeOAuthTokenFamily(family_id)` revokes the entire family.
  - access tokens with family_id (line 87) → also revokes sibling refresh tokens to defeat reuse attacks.
- `writeRevokeOK` (func, line 108) — 200 OK with empty body (RFC 7009 §2.2).
- `writeRevokeError` (func, line 114) — JSON error envelope.

## Imports of note
- `encoding/json`, `log/slog`
- (Reuses helpers from introspect.go.)

## Wired by
- `internal/server/server.go` — mounts `POST /oauth/revoke`.

## Used by
- SDK logout flows; admin lockout via sk_live_* admin token.

## Notes
- ALWAYS 200 OK on missing/invalid token per §2.2 — no oracle for token existence.
- Idempotent: already-revoked tokens still return 200 without re-touching the DB.
- Cross-token-type family revocation: revoking an access token in a rotated family kills the linked refresh tokens — protects against refresh-reuse attacks.
- Audit logged via slog at `oauth.token.revoked`.
