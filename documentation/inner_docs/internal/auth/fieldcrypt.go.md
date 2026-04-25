# fieldcrypt.go

**Path:** `internal/auth/fieldcrypt.go`
**Package:** `auth`
**LOC:** 101
**Tests:** `fieldcrypt_test.go`

## Purpose
AES-256-GCM transparent field-level encryption for sensitive DB columns, with a domain-separated key derived from the server secret.

## Key types / functions
- `fieldEncryptionPrefix="enc::"` (const, line 15) — sentinel that marks a value as encrypted.
- `FieldEncryptor` (type, line 20) — wraps a `cipher.AEAD`.
- `NewFieldEncryptor` (func, line 26) — requires server secret ≥32 chars; key = `SHA-256(secret + "field-encryption")` (32 bytes); builds AES-GCM AEAD.
- `Encrypt` (func, line 52) — empty in → empty out; random nonce; returns `enc::base64(nonce||ciphertext)`.
- `Decrypt` (func, line 69) — passthrough for unprefixed values (transparent migration of pre-encryption rows); strips prefix, base64-decodes, splits nonce, GCM-opens.
- `IsEncrypted` (func, line 99) — prefix sniff.

## Imports of note
- `crypto/aes`, `crypto/cipher` — AES-256-GCM AEAD.
- `crypto/sha256` — domain-separated key derivation.

## Used by
- `internal/storage/sqlite/*.go` — encrypts OAuth tokens, MFA secrets, and other sensitive columns at write/read.

## Notes
- Domain separator `"field-encryption"` ensures this key is distinct from session/JWT keys derived from the same secret.
- Migration-friendly Decrypt: legacy unencrypted rows still load correctly; rotation strategy is "encrypt on next write".
- Nonce reuse is avoided by `crypto/rand` per-call; do not switch to a deterministic nonce.
- AAD is nil — no extra binding to row context.
