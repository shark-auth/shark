# magiclink.go

**Path:** `internal/auth/magiclink.go`
**Package:** `auth`
**LOC:** 391
**Tests:** `magiclink_test.go`

## Purpose
Issuance and verification of email-delivered single-use tokens for three flows: passwordless login (magic link), password reset, and email verification.

## Key types / functions
- `ErrMagicLinkExpired`, `ErrMagicLinkUsed`, `ErrMagicLinkNotFound` (vars, line 21-25).
- `MagicLinkManager` (type, line 28) — wraps store, email sender, session manager, config.
- `NewMagicLinkManager` (func, line 36) — constructor.
- `Sender` / `SetSender` (funcs, line 47-53) — expose/replace the email sender so other managers (e.g. org invites) reuse the wired provider.
- `SendMagicLink` (func, line 58) — 32 random bytes → base64url raw token; stores SHA-256 hash with `mlt_` prefix; renders branded email template; always returns nil to avoid email enumeration.
- `SendPasswordReset` (func, line 135) — same pattern with 15-minute lifetime + `prt_` prefix; redirects to configured front-end URL.
- `VerifyPasswordResetToken` (func, line 196) — hashes raw token, looks up, checks `Used`/`ExpiresAt`, marks used, returns email for caller to set new password.
- `SendEmailVerification` (func, line 228) — 24h lifetime + `evt_` prefix; redirects to `/hosted/default/verify`.
- `VerifyEmailToken` (func, line 287) — same verify pattern, returns email.
- `VerifyMagicLink` (func, line 319) — verifies, marks used, find-or-create user with `email_verified=true`, creates session with `auth_method="magic_link"`.

## Imports of note
- `crypto/sha256` — token hashing for storage.
- `encoding/base64` (RawURLEncoding) — URL-safe token format.
- `internal/email` — branded template rendering + delivery.

## Used by
- `internal/api/auth_handlers.go` — magic link, password reset, verify-email endpoints.
- `internal/api/org_handlers.go` — reuses sender for org invites.

## Notes
- Tokens stored as SHA-256 hash; raw token only ever appears in URL/email.
- Magic link lifetime comes from `cfg.MagicLink.TokenLifetimeDuration()`; password reset hardcoded 15 min; verify hardcoded 24 h.
- Find-or-create on verify (line 353) means a magic link can both sign in existing users and onboard new ones — by design.
- All three "Send..." errors propagate up to the caller; only `SendMagicLink` swallows them silently. Reset/verify return errors so admin code can log them.
