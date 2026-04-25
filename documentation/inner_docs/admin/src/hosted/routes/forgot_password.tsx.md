# forgot_password.tsx

**Path:** `admin/src/hosted/routes/forgot_password.tsx`
**Type:** React route shell
**LOC:** 42

## Purpose
Wires the design `ForgotPasswordForm` to the SharkAuth hosted-login API. Submits the email to the password-reset endpoint and surfaces success/error toasts; the form itself owns the post-submit success banner.

## Exports
- `ForgotPasswordPage` — `{config: HostedConfig}`.

## Props / hooks
- `config: HostedConfig` — provides `app.name`, `app.slug` for routing.
- `useToast()` for success / danger toasts.

## API calls
- POST `/api/v1/auth/password/send-reset-link` `{email}` (cookies included).

## Composed by
- `admin/src/hosted/App.tsx` route table at `/forgot-password`.

## Notes
- Errors are toasted AND re-thrown so the design form can render its inline form-error state.
- `signInHref` derived as `${base}/login`.
