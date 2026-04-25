# email_verify_handlers.go

**Path:** `internal/api/email_verify_handlers.go`
**Package:** `api`
**LOC:** 222
**Tests:** `email_verify_welcome_test.go`

## Purpose
Handlers for the email-verification flow: send verification token (user + admin trigger), verify token, and idempotently fire the welcome email exactly once when verification first succeeds.

## Handlers exposed
- `handleEmailVerifySend` (func, line 19) — `POST /api/v1/auth/email/verify/send`; requires session, returns 200 always (avoids timing leaks); 503 when no `MagicLinkManager` is configured; short-circuits with "already verified" when applicable
- `handleEmailVerify` (func, line 65) — `GET /api/v1/auth/email/verify?token=...`; consumes the token, flips `EmailVerified=true`, then via `MarkWelcomeEmailSent` (UPDATE WHERE welcome_email_sent=0) idempotently sends the welcome email in a detached goroutine
- `handleAdminEmailVerifySend` (func, line 164) — `POST /api/v1/users/{id}/verify/send`; admin-key gated; 404 when user missing, 400 when already verified

## Key types
None — uses inline JSON maps and `email.Message` from `internal/email`.

## Imports of note
- `internal/api/middleware` (mw) — `GetUserID`
- `internal/email` — `RenderWelcome`, `Sender`
- `internal/storage` — `MarkWelcomeEmailSent`, `ResolveBranding`

## Wired by / used by
- User routes mounted at `internal/api/router.go:258–264`
- Admin route at `internal/api/router.go:443`

## Notes
- Welcome-email send is fire-and-forget in a goroutine detached from the request context — `Sender.Send` is synchronous + context-free, so no cancellation plumbing.
- DB errors other than `sql.ErrNoRows` from `MarkWelcomeEmailSent` are silently skipped — a missed welcome isn't worth failing the verify response.
- `dashboardURL` is built from `cfg.Server.BaseURL` + `/admin`; `AppName` is `cfg.MFA.Issuer`.
