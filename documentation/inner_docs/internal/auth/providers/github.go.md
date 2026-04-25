# github.go

**Path:** `internal/auth/providers/github.go`
**Package:** `providers`
**LOC:** 97
**Tests:** `github_test.go` (if present; otherwise covered by oauth_test.go)

## Purpose
GitHub OAuth 2.0 provider implementation conforming to `auth.OAuthProvider`.

## Key types / functions
- `githubEndpoint` (var, line 17) — hardcoded auth/token URLs (annotated `#nosec G101`).
- `githubUserURL` (const, line 22) — `https://api.github.com/user`.
- `GitHub` (type, line 25) — wraps `oauth2.Config`.
- `NewGitHub` (func, line 30) — default scope `["user:email"]`; redirect `<baseURL>/api/v1/auth/oauth/github/callback`.
- `NewGitHubWithConfig` (func, line 48) — test-only constructor that allows endpoint override.
- `Name` (func, line 52) — `"github"`.
- `AuthURL` / `Exchange` (funcs, line 54-60) — standard.
- `GetUser` (func, line 62) — fetches `/user`, decodes `{id, login, name, email, avatar_url}`; falls back to `login` if `name` is empty; converts numeric ID to string.

## Imports of note
- `golang.org/x/oauth2` — flow.

## Used by
- `internal/server/server.go` — registered if `cfg.OAuth.GitHub.Enabled`.

## Notes
- `email` may be empty for users who hide their primary email; production code may need to follow up with `/user/emails` (not currently implemented).
- ProviderID is GitHub's numeric user ID (stable across renames).
