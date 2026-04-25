# session.go

**Path:** `internal/auth/session.go`
**Package:** `auth`
**LOC:** 190
**Tests:** `session_test.go`

## Purpose
Session creation, validation, and HTTP cookie management with HMAC-signed + AES-encrypted cookies via `gorilla/securecookie`.

## Key types / functions
- `cookieName="shark_session"`, `sessionPrefix="sess_"` (consts, line 19-22).
- `ErrSessionNotFound`, `ErrSessionExpired`, `ErrNoCookie` (vars, line 24-28).
- `SessionManager` (type, line 31) — wraps store, codec, lifetime, secure flag.
- `NewSessionManager` (func, line 42) — derives 32-byte hash key (truncated secret) and a separate 32-byte block key via `SHA-256("sharkauth-block-key:" + secret)`; `secure` flag tied to `https://` prefix on baseURL.
- `newSessionID` (func, line 64) — `sess_` + nanoid.
- `CreateSession` (func, line 70) — creates row with `mfa_passed=true`.
- `CreateSessionWithMFA` (func, line 91) — variant with explicit MFA flag.
- `ValidateSession` (func, line 112) — fetches by ID, parses RFC3339 expiry, deletes expired rows on miss.
- `SetSessionCookie` (func, line 135) — writes encrypted cookie (HttpOnly + Lax + dynamic Secure).
- `GetSessionFromRequest` (func, line 155) — decode from cookie.
- `SecureCookies` (func, line 171) — accessor.
- `ClearSessionCookie` (func, line 174) — MaxAge=-1 clear.
- `UpgradeMFA` (func, line 188) — sets `mfa_passed=true` on existing session.

## Imports of note
- `github.com/gorilla/securecookie` — AES + HMAC cookie codec.
- `crypto/sha256` — domain-separated block-key derivation.

## Used by
- `internal/api/auth_handlers.go` — login, logout, MFA upgrade.
- `internal/api/middleware.go` — request session resolution.
- `internal/auth/magiclink.go`, `passkey.go`, `oauth.go` — session creation after authentication.

## Notes
- Block key uses domain separator to prevent key reuse with field encryption / JWT key wrap.
- SameSite=Lax (line 149) — allows top-level cross-site nav (OAuth callbacks) but blocks CSRF on POSTs.
- G124 nosec annotations document why `Secure` is dynamic (local http dev).
