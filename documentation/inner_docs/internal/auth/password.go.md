# password.go

**Path:** `internal/auth/password.go`
**Package:** `auth`
**LOC:** 116
**Tests:** `password_test.go`

## Purpose
Password hashing and verification using Argon2id, with bcrypt fallback to support users imported from Auth0.

## Key types / functions
- `ErrInvalidHash`, `ErrIncompatibleVersion` (vars, line 17-20) — sentinel errors for malformed/legacy hashes.
- `HashPassword` (func, line 24) — argon2id with 16-byte random salt; returns PHC encoded string `$argon2id$v=19$m=...$t=...$p=...$<salt>$<hash>`.
- `VerifyPassword` (func, line 43) — auto-detects bcrypt (`$2a$`/`$2b$`) vs argon2id; constant-time compare via `subtle.ConstantTimeCompare`.
- `NeedsRehash` (func, line 71) — returns true when stored hash is not argon2id (Auth0 import path); callers re-hash on next successful login.
- `isBcryptHash` (func, line 76) — internal prefix sniff.
- `parseArgon2idHash` (func, line 81) — parses PHC string to params + salt + hash; rejects mismatched argon2 version.

## Imports of note
- `golang.org/x/crypto/argon2` — primary KDF.
- `golang.org/x/crypto/bcrypt` — legacy verification only.
- `crypto/subtle` — constant-time compare.

## Used by
- `internal/api/auth_handlers.go` — signup, login, password reset.
- `internal/api/admin_user_handlers.go` — admin create user / set password.

## Notes
- Tunable params come from `config.Argon2idConfig` (memory/iterations/parallelism/saltLength/keyLength).
- Bcrypt path is verify-only — new hashes are always argon2id; `NeedsRehash` lets callers transparently upgrade.
- `keyLength` is read from the stored hash on verify (line 61), not the global config, so old keyLength values still verify.
