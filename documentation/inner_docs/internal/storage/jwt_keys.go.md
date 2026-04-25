# jwt_keys.go

**Path:** `internal/storage/jwt_keys.go`
**Package:** `storage`
**LOC:** 168
**Tests:** `jwt_keys_test.go`

## Purpose
SQLite implementation of JWT signing-key CRUD + rotation, plus the revoked-JTI table used for fast token revocation checks. The `SigningKey` struct itself lives in `storage.go`.

## Interface methods implemented
### JWT signing keys
- `InsertSigningKey` (13)
- `GetActiveSigningKey` (23) — single active key
- `GetActiveSigningKeyByAlgorithm` (31) — multi-algorithm support (ES256, RS256, etc.)
- `GetSigningKeyByKID` (38)
- `RotateSigningKeys` (46) — transactional: marks current actives as `retired` with `rotated_at=now`, then inserts the new active key
- `ListJWKSCandidates` (78) — keys to publish in the JWKS response; includes recently-rotated retired keys when `activeOnly=false` so in-flight tokens stay verifiable

### Revoked JTIs
- `InsertRevokedJTI` (142) — idempotent (`INSERT OR IGNORE`)
- `IsRevokedJTI` (151)
- `PruneExpiredRevokedJTI` (164) — lazy GC; lets the table self-cleanup without a background job

## Tables touched
- jwt_signing_keys (kid, algorithm, public_key_pem, private_key_pem, status, created_at, rotated_at)
- revoked_jti (jti, expires_at)

## Imports of note
- `database/sql`, `time`

## Used by
- `internal/auth/jwt` — signing + JWKS publishing
- `internal/oauth` — token issuance + introspection
- `cmd/shark/cmd/keys.go` rotate-keys CLI

## Notes
- `PrivateKeyPEM` is AES-GCM encrypted at rest by callers (storage just stores the bytes). Public key is plaintext.
- Rotation is fully transactional via `BeginTx` — no window where two keys are simultaneously active.
- Revoked-JTI lookup is intentionally a count rather than a `SELECT 1` so the path is uniform with the count-based stats elsewhere.
