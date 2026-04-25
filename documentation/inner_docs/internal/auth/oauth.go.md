# oauth.go

**Path:** `internal/auth/oauth.go`
**Package:** `auth`
**LOC:** 184
**Tests:** `oauth_test.go`

## Purpose
OAuth provider abstraction and orchestration: maps `OAuthProvider` implementations to a single callback handler that performs find-or-create-user, account linking, and session issuance.

## Key types / functions
- `ErrProviderNotFound`, `ErrProviderNotConfigured` (vars, line 18-21).
- `OAuthProvider` (interface, line 24) — `Name()`, `AuthURL(state)`, `Exchange(ctx, code)`, `GetUser(ctx, token)`.
- `OAuthUserInfo` (struct, line 36) — normalized provider profile (`ProviderID`, `Email`, `Name`, `AvatarURL`).
- `OAuthManager` (type, line 45) — registry of providers + dependencies.
- `NewOAuthManager` (func, line 53) — empty-registry constructor.
- `RegisterProvider` / `GetProvider` (funcs, line 64-75).
- `HandleCallback` (func, line 79) — code → token → userinfo → existing-link lookup → if linked: refresh avatar + new session; else find-or-create user by email and insert `oauth_accounts` row with `oac_` prefix; session `auth_method="oauth:<provider>"`.

## Imports of note
- `golang.org/x/oauth2` — token type (the providers handle the actual flow).
- `github.com/matoous/go-nanoid/v2` — IDs for users and oauth_accounts.

## Used by
- `internal/api/oauth_handlers.go` — `/api/v1/auth/oauth/{provider}/{login,callback}` endpoints.
- `internal/server/server.go` — provider registration at startup.

## Notes
- New users from OAuth are created with `email_verified=true` (provider attests).
- Avatar is updated on every login if provider changed it; avatars-from-other-flows are preserved if first-link path runs but user already has one (line 145).
- Access/refresh tokens are stored in `oauth_accounts` (likely encrypted by `fieldcrypt.go` at the storage layer); not used for any post-link API calls in current code.
