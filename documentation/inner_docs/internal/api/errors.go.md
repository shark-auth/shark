# errors.go

**Path:** `internal/api/errors.go`
**Package:** `api`
**LOC:** 118
**Tests:** `errors_test.go`

## Purpose
Canonical structured-error envelope (W18) for all `/auth/**` and `/admin/**` endpoints. OAuth endpoints under `/oauth/**` deliberately bypass this and use the RFC 6749 §5.2 shape from `internal/oauth/errors.go` so OAuth client SDKs keep working.

## Handlers exposed
None. This file is a helper package surface.

## Key types
- `ErrorResponse` (struct, line 36) — `{error, message, code, docs_url?, details?}`

## Functions
- `NewError(code, message)` (func, line 47) — populates Error+Code from same string
- `(ErrorResponse) WithDocsURL(code)` (method, line 58) — appends `https://docs.shark-auth.com/errors/<code>`
- `(ErrorResponse) WithDetails(map)` (method, line 65) — attaches structured context (e.g. min_length, retry_after)
- `WriteError(w, status, err)` (func, line 73) — JSON-encodes envelope with `Content-Type: application/json`

## Constants (error code catalog, lines 81–118)
- Generic: `CodeInvalidRequest`, `CodeUnauthorized`, `CodeForbidden`, `CodeNotFound`, `CodeConflict`, `CodeRateLimited`, `CodeInternal`
- Credentials: `CodeInvalidCredentials`, `CodeAccountLocked`, `CodeMFARequired`, `CodeSessionExpired`, `CodeTokenUsed`, `CodeInvalidToken`
- Password policy: `CodeWeakPassword`, `CodePasswordTooShort`, `CodePasswordTooCommon`, `CodePasswordInBreach`
- MFA / passkey: `CodeEnrollmentAlreadyComplete`, `CodeChallengeExpired`, `CodeChallengeInvalid`, `CodeNoMatchingCredential`
- Email/magic-link: `CodeEmailAlreadyVerified`, `CodeMagicLinkExpired`
- Admin: `CodeBootstrapLocked`, `CodeInvalidScope`

## Imports of note
- `encoding/json`, `net/http` only — zero internal deps

## Wired by / used by
- Used by W18-aware handlers (e.g. `oauth_handlers.go` redirect_uri validation calls `WriteError(w, 400, NewError(CodeInvalidRequest,...).WithDocsURL(...))`); legacy handlers still emit raw `map[string]string` payloads via `writeJSON`.

## Notes
- `error` and `code` are typically the same; they diverge when one class (e.g. `weak_password`) fans out into refined codes (`password_too_short`, `password_in_breach`).
- `docsURLBase` (`https://docs.shark-auth.com/errors/`) is stable so integrators can switch on `code` today.
