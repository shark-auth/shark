# headers.go

**Path:** `internal/proxy/headers.go`
**Package:** `proxy`
**LOC:** 122
**Tests:** see `headers_test.go`

## Purpose
Owns the canonical header contract between SharkAuth's reverse proxy and trusted upstream services: strips client-supplied identity headers (anti-spoofing) and injects authoritative ones derived from the request's authenticated `Identity`.

## Key types / functions
- `Identity = identity.Identity` — package-local alias.
- Header constants: `HeaderUserID`, `HeaderUserEmail`, `HeaderUserRoles`, `HeaderAgentID`, `HeaderAgentName`, `HeaderAuthMethod`, `HeaderCacheAge`, `HeaderAuthMode`.
- `strippedPrefixes = ["X-User-", "X-Agent-", "X-Shark-"]` — case-insensitive match.
- `hasStrippedPrefix(key string) bool` — defensive uppercasing in case caller bypassed `http.Header` canonicalization.
- `StripIdentityHeaders(h http.Header, trusted []string)` — removes prefixed headers; preserves explicit `trusted` allowlist (canonicalized). Deletes both raw + canonical keys.
- `InjectIdentity(h http.Header, id Identity)` — writes the canonical identity headers; empty fields are deleted (lets upstream distinguish "unset" from ""); `X-Shark-Auth-Mode` mirrors `X-Auth-Method`; `X-Shark-Cache-Age` written as integer seconds when `id.CacheAge > 0`.
- `setOrDelete(h http.Header, key, value string)` — internal helper.

## Imports
- `fmt`, `net/http`, `strings`, `internal/identity`.

## Wired by
- Reverse proxy `Director` / request rewrite path in `internal/proxy/proxy.go`.

## Used by
- Upstream services rely on these headers as the trust source for the authenticated principal — they must NOT re-validate tokens once the proxy has stamped them.

## Notes
- Design comment is explicit: empty allowlist + secure default `StripIncoming=true`. Tests cover the unsafe path.
- `Roles` joined via comma. `AuthMethod` cast from typed enum to string twice (once as `X-Auth-Method`, once as `X-Shark-Auth-Mode`) — kept distinct so the rules engine can tag without colliding with consumer-facing semantics.
