# auth.ts

**Path:** `packages/shark-auth-react/src/core/auth.ts`
**Type:** OAuth 2.1 + PKCE flow primitives
**LOC:** 134

## Purpose
The wire-level OAuth client: builds the `/oauth/authorize` URL with PKCE, exchanges authorization codes (or refresh tokens) for access tokens, with optional DPoP proof header.

## Public API
- `generateCodeVerifier(): string` — 48 random bytes base64url-encoded (~64 chars).
- `generateCodeChallenge(verifier): Promise<string>` — SHA-256 over verifier, base64url.
- `interface AuthFlowResult { url: string; state: string }`
- `startAuthFlow(redirectUrl, authUrl, publishableKey): Promise<AuthFlowResult>`
  - Persists verifier + post-login redirect to sessionStorage.
  - Returns `<authUrl>/oauth/authorize?response_type=code&client_id=<pk>&redirect_uri=<origin>/shark/callback&code_challenge=...&code_challenge_method=S256&state=...`.
- `interface TokenResponse { access_token; refresh_token?; expires_in?; expires_at? }`
- `interface ExchangeOptions { code?; refreshToken?; dpopProof? }`
- `exchangeToken(authUrl, publishableKey, opts): Promise<TokenResponse>`
  - **POST** `<authUrl>/oauth/token` form-encoded.
  - With `code`: sends `grant_type=authorization_code` + `code` + stored `code_verifier` + `redirect_uri`.
  - With `refreshToken`: sends `grant_type=refresh_token` + `refresh_token`.
  - If `dpopProof` provided, sets header `DPoP: <jwt>`.
  - On success calls `setAccessToken(...)` to persist; computes `expiresAt` ms from either `expires_at` (sec) or `expires_in` (sec, default 3600).
- `exchangeCodeForToken(code, authUrl, publishableKey)` — `@deprecated` shim that calls `exchangeToken({ code })`.

## Internal dependencies
- `./storage` — `setCodeVerifier`, `getCodeVerifier`, `setRedirectAfter`, `setAccessToken`

## Used by (consumer-facing)
- `SignIn` and `SignUp` components → `startAuthFlow` then `window.location.href = url` (SignUp appends `&screen_hint=signup`).
- `SharkCallback` → `exchangeCodeForToken` after parsing `?code=` from the URL.
- `SharkProvider.getToken` → `exchangeToken({ refreshToken, dpopProof })` for silent refresh.

## Notes
- Callback URI is hard-coded as `${window.location.origin}/shark/callback` — consumers must mount `<SharkCallback />` at that route.
- Throws `Token exchange failed <status>: <body>` on non-2xx so callers see server validation messages.
- PKCE `S256` only; no plain method.
