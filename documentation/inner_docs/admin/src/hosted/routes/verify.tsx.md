# verify.tsx

**Path:** `admin/src/hosted/routes/verify.tsx`
**Type:** React route shell
**LOC:** 102

## Purpose
Email verification landing route. If a `?token=` query is present, performs the GET-based verification on mount and flips the `EmailVerify` surface between pending → success / error. Provides Resend on failure and Continue (OAuth callback redirect) on success.

## Exports
- `VerifyPage` — `{config: HostedConfig}`.

## Props / hooks
- `useToast()`.
- `useState` once for `token` from `URLSearchParams`.
- `useState<VerifyState>` for `state`, `errorMessage`.
- `useEffect` runs verification once on mount (cancellation flag guards async resolve).

## API calls
- GET `/api/v1/auth/email/verify?token=...` (credentials included).
- POST `/api/v1/auth/email/verify/send` `{}` (resend).

## Composed by
- `admin/src/hosted/App.tsx` route table at `/verify`.

## Notes
- `cancelled` flag prevents state updates after unmount.
- Body errors fall back to "Verification failed — the link may have expired." when no message field is present.
- `onContinue` wires to the OAuth callback URL (or `${base}/login` when no `redirect_uri` configured).
- Resend handler only attached when state isn't `success`.
