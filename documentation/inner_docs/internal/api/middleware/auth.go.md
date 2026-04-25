# middleware/auth.go

**Path:** `internal/api/middleware/auth.go`
**Package:** `middleware`
**LOC:** 258
**Tests:** `auth_jwt_test.go`

## Purpose
End-user authentication middleware family covering JWT (Authorization: Bearer) and session-cookie identification, plus orthogonal gates for email-verified and MFA-passed state. This is the single entry point that translates wire credentials into the canonical user/session context keys used by every handler.

## Context keys + accessors
- `UserIDKey`, `SessionIDKey`, `MFAPassedKey`, `AuthMethodKey`, `claimsKey` (lines 16–25)
- `GetUserID`, `GetSessionID`, `GetMFAPassed`, `GetAuthMethod`, `GetClaims` (lines 29–66)

## Middleware exposed
- `RequireSessionFunc(sm *auth.SessionManager, jwtMgr *jwtpkg.Manager)` (line 77) — primary auth gate. Decision tree (PHASE3.md §2.1):
  1. `Authorization: Bearer <token>` → JWT path (NO fall-through on failure). Refresh tokens used as bearer credential return 401 with `WWW-Authenticate: Bearer error="invalid_token", error_description="refresh token cannot be used as access credential"` (§2.3)
  2. No bearer → cookie path via `sm.GetSessionFromRequest` + `sm.ValidateSession`
  3. Neither → 401 + `WWW-Authenticate: Bearer`
- `OptionalSessionFunc(sm, jwtMgr)` (line 155) — best-effort variant: same decision tree but invalid/missing creds proceed unauthenticated. Used by `/oauth/authorize`, `/oauth/device/verify` to support both logged-in and not.
- `RequireSession(next)` (line 207) — legacy placeholder that only checks `shark_session` cookie presence; superseded by `RequireSessionFunc`.
- `RequireEmailVerifiedFunc(isVerified)` (line 223) — depends on `RequireSessionFunc` upstream; 403 `email_verification_required` if user's email not verified
- `RequireMFA(next)` (line 250) — depends on `RequireSessionFunc` upstream; 403 `mfa_required` if `MFAPassedKey` is false

## Imports of note
- `internal/auth` — `SessionManager`
- `internal/auth/jwt` — `Manager`, `Claims`, `ErrRefreshToken`

## Chain order (typical group in `router.go`)
1. `RequireSessionFunc(sm, JWTManager)` — sets identity context
2. `RequireMFA` — gate when full step-up needed
3. `RequireEmailVerifiedFunc(...)` — gate when verified email needed (e.g. mfa enroll, password change, passkey register)

## Wired by / used by
- Mounted from `internal/api/router.go` across `/auth/me`, `/auth/email`, `/auth/passkey/register*`, `/auth/password/change`, `/auth/mfa/{enroll,verify,disable,recovery-codes}`, `/auth/sessions`, `/auth/consents`, `/auth/revoke`, `/api/v1/organizations/*`, `/vault/connect|callback|connections`

## Notes
- JWT branch sets `SessionID` only when `claims.TokenType == "session"` (access tokens lack a SessionID).
- A belt-and-suspenders refresh-token check at line 104 backs up `ErrRefreshToken` (in case a future Validate path returns success for a refresh token).
