# manager.go

**Path:** `internal/auth/jwt/manager.go`
**Package:** `jwt`
**LOC:** 527
**Tests:** `manager_test.go`

## Purpose
JWT lifecycle: issuance (session and access/refresh modes), validation (with alg-confusion guard), key rotation, RBAC + tier enrichment, and per-request revocation check. Algorithms other than RS256 are rejected at parse to prevent alg-confusion (RFC 8725 §2.1).

## Key types / functions
- Sentinel errors (line 24-34): `ErrExpired`, `ErrInvalidSignature`, `ErrRevoked`, `ErrUnknownKid`, `ErrAlgMismatch`, `ErrRefreshToken`.
- `Claims` (type, line 43) — registered claims plus `MFAPassed`, `SessionID`, `TokenType` (`session|access|refresh`), `Tier`, `Roles`, `Scope` (last three omitempty).
- `Manager` (type, line 54) — wraps config, store, baseURL, server secret.
- `defaultUserTier="free"` (const, line 66).
- `userEnrichment` (type, line 73) + `resolveEnrichment` (func, line 87) — single source of truth for tier+roles baking; tier comes from user metadata JSON via `userTierFromMetadata` (line 114), best-effort.
- `NewManager` (func, line 132).
- `issuer` (func, line 142) — uses `cfg.Issuer` or falls back to baseURL.
- `SetCheckPerRequest` (func, line 151) — toggle revocation check at runtime.
- `EnsureActiveKey` (func, line 157) — auto-generates RSA keypair on first boot if none exists.
- `GenerateAndStore` (func, line 171) — generate keypair, compute KID, encode + AES-GCM encrypt private key, optionally rotate.
- `newJTI` (func, line 212) — 16 random bytes → base64url.
- `issueToken` (func, line 221) — fetches active key, decrypts private key, signs RS256 with `kid` header.
- `IssueSessionJWT` (func, line 244) — single token for `mode="session"`, 30d TTL, bakes tier+roles for fast downstream auth (proxy rules, admin handlers).
- `IssueAccessRefreshPair` (func, line 283) — separate access (with enrichment) + refresh (lean) tokens with config TTLs.
- `Refresh` (func, line 351) — validates refresh token; **always** checks revocation (one-time-use) regardless of `check_per_request`; revokes old JTI before issuing new pair.
- `Validate` / `validateInternal` (funcs, line 385-467) — peek alg → peek kid → fetch signing key → verify signature + standard claims (issuer, audience, leeway) → enforce token-type → optional revocation.
- `peekAlg` (func, line 471) — header-only decode that rejects `none`, all `HS*`, and anything other than RS256/ES256.
- `peekKID` (func, line 505) — extracts `kid` from header for key lookup.

## Imports of note
- `github.com/golang-jwt/jwt/v5` — parsing/signing.
- `internal/storage` — signing keys + revoked JTI tables + RBAC.

## Used by
- `internal/api/middleware.go` — request validation.
- `internal/api/auth_handlers.go` — issuance after login.
- `internal/api/admin_jwt_handlers.go` — rotation endpoints.
- `internal/proxy/*` — tier/role authorization on enriched claims.

## Notes
- Refresh tokens are **one-time-use** (rotation pattern) — replay yields `ErrRevoked`.
- `peekAlg` is the alg-confusion firewall; the library parser is also constrained via `WithValidMethods(["RS256"])` (line 421) — defense in depth.
- Enrichment failures **do not** block issuance — JWT goes out with tier defaulted, roles empty (line 260-261).
- `Tier`/`Roles` are baked in at issue time and stale until next refresh — by design (avoids per-request RBAC lookup).
- Lazy pruning of revoked JTI rows on every check (line 365, 456).
