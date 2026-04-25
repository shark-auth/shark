# redirect.go

**Path:** `internal/auth/redirect/redirect.go`
**Package:** `redirect`
**LOC:** 194
**Tests:** `redirect_test.go`

## Purpose
OAuth 2.1 redirect URI validation supporting exact match, wildcard subdomain (`https://*.foo.com`), and loopback (RFC 8252 §8.3) patterns. Pure package — no storage/config/net.http dependencies.

## Key types / functions
- `ErrNotAllowed`, `ErrInvalidURL` (vars, line 13-18) — sentinels.
- `Kind` (type, line 21) + `KindCallback`, `KindLogout`, `KindOrigin` (consts, line 23-27) — selects which allowlist to validate against.
- `Application` (struct, line 32) — pure DTO carrying allowlists, intentionally separated from `storage.Application` to avoid import cycles.
- `Validate` (func, line 53) — main entry point. Steps: parse URL → reject empty/whitespace scheme → reject userinfo → reject fragment (OAuth 2.1 §3.1.2) → select allowlist → normalise → linear scan with exact / wildcard / loopback rules.
- `normalize` (func, line 144) — lowercase scheme+host, strip default ports (`:80`/`:443`), strip trailing `/` when path is just `/`.
- `stripPort` (func, line 170) — host:port splitter that handles IPv6 literals.
- `matchWildcardSubdomain` (func, line 186) — exactly one label between host and base domain (no nested wildcards).

## Imports of note
- `net/url`, `strings`, `errors` — standard library only.

## Used by
- `internal/api/oauth_handlers.go` and `internal/api/auth_handlers.go` — validating `redirect_uri` and post-logout redirects.
- `internal/api/admin_app_handlers.go` — validating allowlist entries on app create/update.

## Notes
- Wildcard supports `https://*.foo.com` only (single-label subdomain match), no `https://*.foo.com/path/*` path wildcards.
- Loopback rule allows any port for `http://127.0.0.1` and `http://localhost` per RFC 8252 §8.3 (native app dev).
- Malformed allowlist patterns are silently skipped (line 92), not errored — by design so a single bad row doesn't break the whole app.
