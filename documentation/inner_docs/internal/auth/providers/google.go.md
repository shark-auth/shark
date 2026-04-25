# google.go

**Path:** `internal/auth/providers/google.go`
**Package:** `providers`
**LOC:** 78
**Tests:** `google_test.go` (if present; otherwise covered by oauth_test.go)

## Purpose
Google OAuth 2.0 provider implementation conforming to `auth.OAuthProvider`.

## Key types / functions
- `googleUserInfoURL` (const, line 16) — `https://www.googleapis.com/oauth2/v2/userinfo`.
- `Google` (type, line 19) — wraps `oauth2.Config`.
- `NewGoogle` (func, line 24) — defaults to scopes `[openid, email, profile]`; redirect URL `<baseURL>/api/v1/auth/oauth/google/callback`.
- `Name` (func, line 40) — `"google"`.
- `AuthURL` (func, line 42) — appends `access_type=offline` (refresh-token grant).
- `Exchange` (func, line 46) — standard `oauth2.Config.Exchange`.
- `GetUser` (func, line 50) — fetches v2 userinfo, decodes `{id, email, name, picture}` into `OAuthUserInfo`.

## Imports of note
- `golang.org/x/oauth2` + `oauth2/google` — Google endpoint constants.

## Used by
- `internal/server/server.go` — registered with `OAuthManager` if `cfg.OAuth.Google.Enabled`.

## Notes
- Uses the v2 userinfo endpoint (legacy but stable). Could move to OIDC discovery if more claims needed.
- `access_type=offline` issues a refresh_token only on first consent; subsequent logins return only access_token.
