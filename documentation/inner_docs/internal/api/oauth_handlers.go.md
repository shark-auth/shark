# oauth_handlers.go

**Path:** `internal/api/oauth_handlers.go`
**Package:** `api`
**LOC:** 228
**Tests:** `oauth_handlers_test.go`, `oauth_redirect_integration_test.go`

## Purpose
Social-login (OAuth client) flow — Start (redirect to provider) and Callback (exchange code, link/create user, mint session, optionally redirect to caller). NOTE: this is the OAuth **client** for "Login with Google/GitHub/Apple/Discord" — the OAuth Authorization Server lives at `internal/oauth/` and is mounted at `/oauth/*` from `router.go`.

## Handlers exposed
- `handleOAuthStart` (func, line 27) — `GET /api/v1/auth/oauth/{provider}`; mints 16 random bytes as state, sets short-lived `shark_oauth_state` cookie (HttpOnly, SameSite=Lax, Secure mirrors session cookies, MaxAge=5 min), redirects 302 to provider AuthURL
- `handleOAuthCallback` (func, line 68) — `GET /api/v1/auth/oauth/{provider}/callback`; constant-time state compare, clears state cookie, exchanges code via `OAuthManager.HandleCallback` (which finds/creates user + session row), fires `AuthFlowTriggerOAuthCallback`, sets session cookie, validates `redirect_uri` against the default application's `AllowedCallbackURLs` via `redirect.Validate`, optionally issues JWT (session or access/refresh)
- `initOAuthManager` (method, line 204) — registers Google/GitHub/Apple/Discord providers when their client IDs are configured

## Key constants
- `oauthStateCookieName = "shark_oauth_state"` (line 21)
- `oauthStateTTL = 5 * time.Minute` (line 22)

## Imports of note
- `internal/auth` — `OAuthManager`
- `internal/auth/providers` — Google/GitHub/Apple/Discord
- `internal/auth/redirect` — `Validate` for callback allowlist
- `internal/storage` — `GetDefaultApplication`, `AuthFlowTriggerOAuthCallback`

## Wired by / used by
- Routes registered in `internal/api/router.go:267–270`
- `initOAuthManager` invoked by `NewServer` at `router.go:166`

## Notes
- State cookie's `Secure` flag is dynamic (tied to `base_url` scheme via `SessionManager.SecureCookies()`) so local http dev still works — `#nosec G124` annotated.
- `OAuthManager.HandleCallback` already creates the session; the handler doesn't recreate it. On a flow block/redirect, the cookie is NOT set.
- Apple registration only requires `ClientID` + `TeamID` (no `ClientSecret`) — Apple uses a JWT client assertion built elsewhere.
