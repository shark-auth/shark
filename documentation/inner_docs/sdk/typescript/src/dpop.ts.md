# dpop.ts

**Path:** `sdk/typescript/src/dpop.ts`
**Type:** RFC 9449 DPoP signer
**LOC:** 262

## Purpose
Generate, load, and emit RFC 9449 DPoP proof JWTs bound to an ECDSA P-256 keypair. Consumers wire the prover into `SharkClient`, `DeviceFlow`, or `exchangeToken` to bind their tokens.

## Public API
- `class DPoPProver`
  - `static generate(): Promise<DPoPProver>` — fresh ES256 keypair via `jose.generateKeyPair`
  - `static fromPem(pem: string | Buffer): Promise<DPoPProver>` — load existing PKCS#8 PEM
  - `get jkt: string` — RFC 7638 JWK SHA-256 thumbprint (base64url) — use as `dpop_jkt` in authorization requests
  - `get publicJwk: EcPublicJwk` — copy of public key (no private material)
  - `privateKeyPem(): Promise<string>` — export PKCS#8 PEM for persistence
  - `createProof(opts: CreateProofOptions): Promise<string>` — compact JWT proof
- `interface CreateProofOptions`
  - `method: string` — uppercased into `htm`
  - `url: string` — query/fragment stripped per §4.2 → `htu`
  - `nonce?: string` — server-supplied DPoP nonce
  - `accessToken?: string` — adds `ath = base64url(SHA-256(token))`
  - `iat?`, `jti?` — test overrides

## Wiring keys
- New install: `await DPoPProver.generate()` → store `await prover.privateKeyPem()` securely
- Reload: `await DPoPProver.fromPem(pem)`
- Pass into `new SharkClient({ dpopProver })` or `new DeviceFlow({ dpopProver })`

## Internal dependencies
- `jose` — `generateKeyPair`, `importPKCS8`, `exportJWK`, `exportPKCS8`, `SignJWT`
- `errors.ts` — `DPoPError` for invalid keys / args

## Header
- JWT `typ`: `dpop+jwt`
- JWT `alg`: `ES256`
- JWT `jwk`: embedded public key

## Notes
- Key must be ECDSA P-256; other curves throw `DPoPError`.
- `jti` is generated via crypto random (not exposed publicly) when not overridden.
- ath/nonce/jti claims appended only when provided to `createProof`.
