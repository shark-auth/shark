# keys.go

**Path:** `internal/auth/jwt/keys.go`
**Package:** `jwt`
**LOC:** 209
**Tests:** `keys_test.go`

## Purpose
RSA keypair generation, KID derivation, PEM encode/decode, AES-GCM encryption of private-key PEM with a domain-separated key derived from the server secret, and JWK serialization.

## Key types / functions
- `generateRSAKeypair` (func, line 17) — 2048-bit RSA via `crypto/rand`.
- `computeKID` (func, line 22) — base64url(SHA-256(DER public key))[:16].
- `encodePublicKeyPEM` (func, line 32) — PKIX PEM block.
- `encodePrivateKeyPEM` (func, line 43) — PKCS#8 PEM block (returns plaintext bytes; caller must `encryptPEM`).
- `deriveAESKey` (func, line 54) — `SHA-256(secret + "jwt-key-encryption")` — domain separator prevents reuse with session-cookie or field-encryption keys.
- `encryptPEM` (func, line 61) — AES-GCM seal with random nonce; returns `base64(nonce||ciphertext)`; wipes plaintext slice before return.
- `EncryptPEM` / `DecryptPEM` (funcs, line 89-97) — exported versions for OAuth server.
- `decryptPEM` (func, line 101) — base64 decode → AES-GCM open.
- `decryptPrivateKey` (func, line 127) — decrypts and parses to `*rsa.PrivateKey`; defers a wipe of intermediate bytes.
- `decodePublicKeyPEM` (func, line 156) — parses stored public-key PEM (plaintext column).
- `jwkFromPublicKey` (func, line 173) — RFC 7517 JWK map: `{kty:RSA, use:sig, alg:RS256, kid, n, e}` (base64url, no padding).
- `pubKeyFromJWK` (func, line 195) — test-only inverse of `jwkFromPublicKey`.

## Imports of note
- `crypto/rsa`, `crypto/aes`, `crypto/cipher`, `crypto/sha256`, `crypto/x509`, `encoding/pem` — all standard.

## Used by
- `internal/auth/jwt/manager.go` — issuance/validation paths.
- `internal/api/jwks_handlers.go` — `/.well-known/jwks.json` route via `jwkFromPublicKey`.

## Notes
- Private keys are stored encrypted at rest; the AES key never persists — derived from server secret on demand.
- Plaintext PEM bytes are zeroed in `encryptPEM` (line 80) and via `defer` in `decryptPrivateKey` (line 132).
- KID truncation to 16 chars is unique enough for the small key set typical of a single tenant; collision risk is negligible.
- ES256 keypair handling lives in `es256.go`.
