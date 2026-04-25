# login.tsx

**Path:** `admin/src/hosted/routes/login.tsx`
**Type:** React route shell
**LOC:** 199

## Purpose
The hosted-login front door. Bridges the design `SignInForm` to all wired auth methods: password login, magic-link request, passkey (full WebAuthn handshake), and OAuth provider redirect — with consistent post-success redirect to the requesting OAuth `redirect_uri` (and `state` round-trip).

## Exports
- `LoginPage` — `{config: HostedConfig}`.
- WebAuthn helpers: `base64urlDecode`, `arrayBufferToBase64url`, `serializeAssertion`.

## Props / hooks
- `config: HostedConfig` — auth methods, OAuth params (client_id, redirect_uri, state, scope), app name + slug.
- `useLocation` from `wouter` for in-app navigation.
- `useToast()` for danger toasts.

## API calls
- POST `/api/v1/auth/login` `{email, password}`
- POST `/api/v1/auth/magic-link/send` `{email}`
- POST `/api/v1/auth/passkey/login/begin` then POST `/api/v1/auth/passkey/login/finish`
- GET (redirect) `/api/v1/auth/oauth/{providerID}?client_id&state&redirect_uri&scope`

## Composed by
- `admin/src/hosted/App.tsx` route table at `/login`.

## Notes
- `mfaRequired` response branch navigates to `/mfa?method=totp` instead of redirecting to OAuth.
- After successful login the page builds the OAuth callback URL and assigns `window.location.href` (full reload, not SPA nav) so the consumer app's session cookie is honoured.
- Passkey path falls back to `/passkey` route when `window.PublicKeyCredential` is unavailable.
- `serializeAssertion` mirrors the WebAuthn `AuthenticatorAssertionResponse` into a base64url-encoded JSON payload the server expects.
