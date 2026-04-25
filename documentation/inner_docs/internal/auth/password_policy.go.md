# password_policy.go

**Path:** `internal/auth/password_policy.go`
**Package:** `auth`
**LOC:** 53
**Tests:** `password_policy_test.go`

## Purpose
Password complexity validation and a small embedded blocklist of the most common weak passwords.

## Key types / functions
- `commonPasswords` (var, line 10) — `map[string]bool` of ~24 popular passwords (`password`, `12345678`, `qwerty123`, etc.).
- `ValidatePasswordComplexity` (func, line 27) — checks `len(password) >= minLength`, blocklist (lowercased), and presence of at least one uppercase, lowercase, and digit; returns empty string on success or human-readable rejection reason.

## Imports of note
- `strings`, `unicode` — only the standard library.

## Used by
- `internal/api/auth_handlers.go` — signup and password change.
- `internal/api/admin_user_handlers.go` — admin set-password.

## Notes
- Blocklist intentionally tiny (binary-size constraint); HIBP/zxcvbn integration would be a future upgrade.
- Returns a string rather than an error — callers surface the message verbatim to clients.
- Special characters not required (only U/L/D classes).
