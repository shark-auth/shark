# signing_keys.tsx

**Path:** `admin/src/components/signing_keys.tsx`
**Type:** React component (page)
**LOC:** ~400

## Purpose
JWT signing key management—view active keys, rotation history, JWK endpoint, algorithm settings, key expiration.

## Exports
- `SigningKeys()` (default) — function component

## Features
- **Key list** — algorithm (RS256|ES256|EdDSA), key ID, created, expires, status (active|archived)
- **Rotation** — manual key rotation, automatic rotation scheduling
- **JWK set** — endpoint for public key discovery (/.well-known/jwks.json)
- **Algorithm selection** — choose signing algorithm
- **Key details** — public key preview, fingerprint, usage stats
- **Expiration** — set key lifetime, archive old keys

## Hooks used
- `useAPI('/admin/signing-keys')` — list keys

## State
- `selected` — current key
- `rotateOpen` — rotation confirmation

## API calls
- `GET /api/v1/admin/signing-keys` — list keys
- `POST /api/v1/admin/signing-keys/rotate` — rotate key
- `PATCH /api/v1/admin/signing-keys/{id}` — update expiration
- `GET /.well-known/jwks.json` — public keys (unauthenticated)

## Composed by
- App.tsx

## Notes
- JWT keys are sensitive; only admins can view
- Public endpoint (JWK set) is unauthenticated for OAuth clients
- Expiration managed server-side; old keys auto-archived
- Rotation is zero-downtime (both old and new keys valid briefly)
