# passkey.tsx

**Path:** `admin/src/hosted/routes/passkey.tsx`
**Type:** React route shell
**LOC:** 137

## Purpose
Standalone passkey-only sign-in page. Performs the full WebAuthn `navigator.credentials.get` handshake against the SharkAuth passkey endpoints and redirects to the OAuth callback on success.

## Exports
- `PasskeyPage` — `{config: HostedConfig}`.
- WebAuthn helpers: `base64urlDecode`, `arrayBufferToBase64url`, `serializeAssertion`.

## Props / hooks
- `useLocation` (from `wouter`) for backup MFA navigation.
- `useToast()`.

## API calls
- POST `/api/v1/auth/passkey/login/begin` `{}`
- POST `/api/v1/auth/passkey/login/finish` (serialized assertion).

## Composed by
- `admin/src/hosted/App.tsx` route table at `/passkey`.

## Notes
- Browser support guard: shows a danger toast if `window.PublicKeyCredential` is missing.
- `mfaRequired` response navigates to `/mfa?method=totp`.
- Custom layout (not the design `Card`) — heading, paragraph, button, back-to-sign-in link styled inline so this fallback page can survive without a tenant theme.
- Helper functions duplicated here and in `login.tsx` — candidate for extraction into a shared module.
