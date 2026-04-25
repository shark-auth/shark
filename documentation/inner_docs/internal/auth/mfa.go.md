# mfa.go

**Path:** `internal/auth/mfa.go`
**Package:** `auth`
**LOC:** 175
**Tests:** `mfa_test.go`

## Purpose
TOTP enrollment + validation and bcrypt-hashed recovery code generation/redemption.

## Key types / functions
- `MFAManager` (type, line 30) — wraps `storage.Store` + `config.MFAConfig`.
- `NewMFAManager` (func, line 36) — constructor.
- `GenerateSecret` (func, line 45) — creates a TOTP key via `pquerna/otp/totp` (SHA-1, 6 digits, 30s); returns base32 secret + `otpauth://` URI for QR.
- `ValidateTOTP` (func, line 67) — verifies code with skew=1 (one 30s step tolerance).
- `GenerateRecoveryCodes` (func, line 79) — clears existing codes, generates N (default 10) 8-char alphanumeric codes, bcrypt-hashes each (cost 10), stores hashes, returns plaintext to caller (one-time display).
- `VerifyRecoveryCode` (func, line 123) — normalizes input (lowercase + strip dashes), iterates active codes, marks matched code used; constant-time dummy compare on miss to avoid timing oracle.
- `generateRandomCode` (func, line 158) — rejection-sampling alphanumeric generator (avoids modulo bias).

## Imports of note
- `github.com/pquerna/otp` + `pquerna/otp/totp` — TOTP RFC 6238.
- `golang.org/x/crypto/bcrypt` — recovery-code hashing (cost 10).
- `github.com/matoous/go-nanoid/v2` — recovery code row IDs (`mrc_` prefix).

## Used by
- `internal/api/mfa_handlers.go` — enrollment, verify, recovery flows.
- `internal/api/auth_handlers.go` — login completion path.

## Notes
- Issuer falls back to `"SharkAuth"` if config empty (line 47).
- Recovery code count defaults to 10 if config is zero/negative (line 81).
- Constant-time dummy on miss path (line 150) is intentional to keep "no codes" and "no match" timing indistinguishable.
- Per-row bcrypt comparison is O(n) over codes; acceptable since N=10.
