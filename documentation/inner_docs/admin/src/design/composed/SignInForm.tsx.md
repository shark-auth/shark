# SignInForm.tsx

**Path:** `admin/src/design/composed/SignInForm.tsx`
**Type:** React component (composed surface)
**LOC:** 368

## Purpose
The full hosted-login surface. Renders any combination of password, magic-link, passkey, and OAuth-provider auth methods based on tenant config, plus links to sign-up and forgot-password. Owns its own validation, focus management, and per-method loading state.

## Exports
- `SignInForm` — see props below.
- `SignInFormProps` (interface).

## Props / hooks
- `appName: string` — heading.
- `authMethods: ('password' | 'magic_link' | 'passkey' | 'oauth')[]` — set of enabled methods.
- `oauthProviders?: {id, name, iconUrl?}[]`.
- Handlers: `onPasswordSubmit(email, password)`, `onMagicLinkRequest(email)`, `onPasskeyStart()`, `onOAuthStart(providerID)`.
- Navigation: `signUpHref?`, `forgotPasswordHref?`.
- Local state: `email`, `password`, errors per field, `formError`, `loading`, `magicLoading`, `passkeyLoading`.

## API calls
- None — all wiring through callback props (route shells call `/api/v1/auth/login`, `/auth/magic-link/send`, `/auth/passkey/login/begin`, etc.).

## Composed by
- `admin/src/hosted/routes/login.tsx` (`LoginPage`).
- React SDK helpers that surface the form in consumer apps.

## Notes
- Validation: trimmed-required + email regex; magic-link path validates only email.
- Focus management: invalid email focuses `signin-email`, otherwise `signin-password`. Form-level errors also re-focus the email input.
- Imports raw asset paths (`sharky-glyph.png`, `sharky-wordmark.png`) for branding header.
- Hanken Grotesk font hard-coded for the brand title; everything else uses `tokens.type.body.family`.
