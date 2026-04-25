# auth_handlers.go

**Path:** `internal/api/auth_handlers.go`
**Package:** `api`
**LOC:** 627
**Tests:** `auth_handlers_test.go`

## Purpose
HTTP handlers for the core password-auth surface: signup, login, logout, /me, password reset/change, plus shared helpers (`writeJSON`, `userResponseMap`, `recordLoginFailure`, `emailRegex`).

## Handlers exposed
- `handleSignup` (func, line 78) — `POST /api/v1/auth/signup`; validates email regex, password complexity (configurable min length, default 8), checks for existing email, Argon2id-hashes password, creates user `usr_<nanoid>`, fires `AuthFlowTriggerSignup`, mints session + sets cookie, optionally issues JWT (session or access/refresh pair), emits `WebhookEventUserCreated`
- `handleLogin` (func, line 206) — `POST /api/v1/auth/login`; checks `LockoutManager` (5 fails / 15 min), looks up user by email, verifies password (constant-time leak-resistant errors), rehashes if `auth.NeedsRehash` (e.g. bcrypt from Auth0 migration), fires `AuthFlowTriggerLogin`, creates session with `mfaPassed=false` when MFA is enabled, returns `{mfaRequired: true}` to gate JWT issuance
- `handleLogout` (func, line 336) — `POST /api/v1/auth/logout`; revokes JWT JTI when authMethod=jwt, deletes session, clears cookie
- `handleMe` (func, line 356) — `GET /api/v1/auth/me`; returns the authenticated user (fresh DB read)
- `handlePasswordResetSend` (func, line 378) — `POST /api/v1/auth/password/send-reset-link`; always returns 200 to avoid email-enumeration, sends reset email if the user exists
- `handlePasswordReset` (func, line 421) — `POST /api/v1/auth/password/reset`; verifies token, validates new password complexity, rotates hash, fires `AuthFlowTriggerPasswordReset`
- `handleChangePassword` (func, line 502) — `POST /api/v1/auth/password/change`; requires session + current password (when one exists), rotates to argon2id

## Helpers
- `recordLoginFailure` (line 589) — emits `user.login` audit row with `status=failure`
- `writeJSON` (line 608) — JSON encode + `Content-Type: application/json`
- `userResponseMap` (line 616) — flatten `userResponse` for JWT field merging

## Key types
- `signupRequest` (struct, line 24), `loginRequest` (line 31), `passwordResetSendRequest` (line 37), `passwordReset` (line 42), `changePasswordRequest` (line 48), `userResponse` (line 54)
- `emailRegex` (var, line 21) — simple email validator

## Imports of note
- `internal/auth` — password hashing, lockout, complexity validation
- `internal/api/middleware` (mw) — context user ID + auth method
- `internal/storage` — user persistence + webhook event constants
- `gonanoid` — user ID generator

## Wired by / used by
- Routes registered in `internal/api/router.go:241–311`

## Notes
- Argon2id is the canonical hash; bcrypt accepted on first login then upgraded.
- All password-policy responses use the `weak_password` error code — fine-grained codes (`password_too_short` etc.) live in `errors.go`.
- Lockout is in-memory (per-process) — multi-replica deploys are NOT covered.
