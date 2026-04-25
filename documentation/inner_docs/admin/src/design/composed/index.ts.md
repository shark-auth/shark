# index.ts

**Path:** `admin/src/design/composed/index.ts`
**Type:** Barrel re-export
**LOC:** 29

## Purpose
Public entry point for the `design/composed` namespace. Re-exports each composed auth surface (form/page-level component) plus its prop types so consumers can import either directly from `composed/index` or treat it as the design-system facade.

## Exports
- Components: `SignInForm`, `SignUpForm`, `ForgotPasswordForm`, `ResetPasswordForm`, `MFAForm`, `PasskeyButton`, `OAuthProviderButton`, `MagicLinkSent`, `EmailVerify`, `ErrorPage`.
- Types: `SignInFormProps`, `SignUpFormProps`, `ForgotPasswordFormProps`, `ResetPasswordFormProps`, `MFAFormProps`, `PasskeyButtonProps`, `OAuthProviderButtonProps`, `MagicLinkSentProps`, `EmailVerifyProps`, `ErrorPageProps`, `ErrorPageAction`.

## Composed by
- `admin/src/hosted/routes/*.tsx` (route wrappers import named composed components from here or directly from their files).
- External React SDKs that lift these surfaces into consumer apps.

## Notes
- No runtime logic — pure re-export module.
- Order in the file mirrors the auth-flow journey (sign in → sign up → reset → MFA → passkey → OAuth → magic-link confirmation → verify → error).
