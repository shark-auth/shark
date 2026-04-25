# oidc.go

**Path:** `internal/sso/oidc.go`
**Package:** `sso`
**LOC:** 152
**Tests:** `oidc_test.go`

## Purpose
OIDC authentication flow: constructs authorization URLs, exchanges codes, validates ID tokens (with nonce), creates/links users, establishes sessions. Shark consumes upstream OIDC IdPs (NOT Shark as IdP).

## Key types / functions
- `OIDCState` (struct, line 18) — in-progress auth flow state (ConnectionID, State, Nonce)
- `SSOManager.BeginOIDCAuth()` (line 26) — returns authorization URL + state/nonce tokens for redirect
- `SSOManager.HandleOIDCCallback()` (line 62) — exchanges code, validates ID token (nonce check), extracts claims, finds/creates user, creates session
- `SSOManager.oidcOAuth2Config()` (line 135) — builds oauth2.Config from connection config
- `randomToken()` (line 146) — generates 16-byte random hex state/nonce tokens

## Imports of note
- `github.com/coreos/go-oidc/v3/oidc` — OpenID Connect provider discovery + token verification
- `golang.org/x/oauth2` — authorization code flow, token exchange

## Wired by
- `internal/api/sso_handlers.go` (GET authorize, GET callback endpoints)
- SSOManager.GetConnection (loads connection config from storage)

## Used by
- Web login flow: user clicks "login with OIDC", frontend redirects to BeginOIDCAuth result, returns to callback handler

## Notes
- Nonce verified in ID token claims to prevent replay attacks (line 97).
- State + Nonce must be stored server-side across request boundary; currently in-memory map (line 18 comment).
- Supports sub (subject), email, name claims; email is required (line 114).
- CallbackURL auto-constructed from BaseURL + connectionID (line 136).
- findOrCreateUser handles user lookup/creation and SSO identity linking (line 118).
- Session created with auth method "sso" (line 126).
