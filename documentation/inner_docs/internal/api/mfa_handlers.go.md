# mfa_handlers.go

**Path:** `internal/api/mfa_handlers.go`
**Package:** `api`
**LOC:** 475
**Tests:** `mfa_handlers_test.go`, `mfa_pending_test.go`

## Purpose
Handlers for the user-facing TOTP MFA lifecycle: enroll (generate secret + QR), verify enrollment (and auto-mint recovery codes), challenge during login (upgrade partial session), recovery-code login, disable, and regenerate recovery codes.

## Handlers exposed
- `handleMFAEnroll` (func, line 41) — `POST /api/v1/auth/mfa/enroll`; requires fully-authed session; 409 when MFA already verified; generates TOTP secret via `auth.NewMFAManager`, encrypts with `FieldEncryptor`, stores on user, returns `{secret, qr_uri}`
- `handleMFAVerify` (func, line 109) — `POST /api/v1/auth/mfa/verify`; validates the first TOTP code, sets `MFAEnabled/MFAVerified/MFAVerifiedAt`, generates + returns recovery codes
- `handleMFAChallenge` (func, line 205) — `POST /api/v1/auth/mfa/challenge`; partial-session only (`mfa_passed=false`); validates TOTP, calls `SessionManager.UpgradeMFA(sessionID)`
- `handleMFARecovery` (func, line 285) — `POST /api/v1/auth/mfa/recovery`; partial-session only; verifies recovery code via `mfaMgr.VerifyRecoveryCode`, upgrades session
- `handleMFADisable` (func, line 355) — `DELETE /api/v1/auth/mfa`; requires current TOTP code; clears secret/verified/verified_at, deletes recovery codes
- `handleMFARecoveryCodes` (func, line 435) — `GET /api/v1/auth/mfa/recovery-codes`; regenerates and returns codes (replaces existing set)

## Key types
- `mfaEnrollResponse` (struct, line 13)
- `mfaVerifyRequest` / `mfaChallengeRequest` / `mfaRecoveryRequest` / `mfaDisableRequest` (lines 19–36)

## Imports of note
- `internal/auth` — `MFAManager`, `ValidateTOTP`, `GenerateRecoveryCodes`, `VerifyRecoveryCode`
- `internal/api/middleware` (mw) — context user ID, session ID, MFA-passed flag

## Wired by / used by
- Routes registered in `internal/api/router.go:315–332` — challenge/recovery in a session-only group, enroll/verify/disable/recovery-codes additionally gated by `RequireMFA` + `requireVerified`

## Notes
- Re-enroll is allowed while `MFAVerifiedAt == nil` (pending secret) so a user can re-scan their QR; once verified, re-enroll requires disable first.
- TOTP secrets are stored encrypted at-rest via `FieldEncryptor` (envelope: `cfg.Server.Secret`).
- `handleMFAChallenge` and `handleMFARecovery` deliberately don't gate on email-verified — they fire before MFA passes during login.
