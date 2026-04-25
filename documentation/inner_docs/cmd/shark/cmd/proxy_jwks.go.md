# proxy_jwks.go

**Path:** `cmd/shark/cmd/proxy_jwks.go`
**Package:** `cmd`
**LOC:** 404
**Tests:** proxy_jwks_test.go

## Purpose
JWKS-aware bearer-token verifier originally written for the standalone `shark proxy` mode. Fetches and caches `/.well-known/jwks.json` from the auth server, refreshes on a schedule and on kid-miss, and verifies RS256/ES256 JWTs into a `proxy.Identity`.

## Key types / functions
- `jwksCache` (struct, line 34) — base URL, HTTP client, refresh interval, expected audiences/issuer (W15c), kid→key map, singleflight for refresh coalescing, kid-miss rate limit.
- `jwksDoc`, `jwksKey` (structs, lines 68-84) — JWKS wire types.
- `newJWKSCache` (func, line 86) — defaults: 15min refresh, 10s min refresh interval.
- `Start` (method, line 99) — synchronous initial fetch + background refresh goroutine.
- `refresh` (method, line 120) — 1 MiB body cap to defeat OOM amplification.
- `keyForKid` (method, line 182) — read-locked cache lookup, rate-limited + singleflighted refresh on miss.
- `parseJWK` (func, line 219) — decodes RSA + EC P-256/384/521 keys.
- `verifyBearer` (method, line 272) — parses Bearer JWT, enforces RS256/ES256, checks issuer (via golang-jwt) and audience (manual set-intersection because `jwtlib.WithAudience` only supports a single expected value).
- `extractAudiences` (func, line 365) — handles aud as string or []string per RFC 7519 §4.1.3.
- `audienceMatches` (func, line 395).

## Imports of note
- `github.com/golang-jwt/jwt/v5`
- `golang.org/x/sync/singleflight`
- `internal/proxy` (for `proxy.Identity`).
- `crypto/rsa`, `crypto/ecdsa`, `crypto/elliptic`.

## Wired by / used by
- Standalone proxy verifier; not directly registered on the cobra root (the standalone `shark proxy` was deprecated, but the JWKS verifier code remained).

## Notes
- W15c hardening notes throughout: rate limiting, singleflight refresh, body size cap, manual audience set-intersection.
- Audience check loops over expected audiences because `jwt/v5`'s `WithAudience` requires ALL match — wrong semantics here.
