# tokens.ts

**Path:** `sdk/typescript/src/tokens.ts`
**Type:** Agent token verify + decode
**LOC:** 144

## Purpose
Verify (signature + standard claims via JWKS) or decode (no verify) Shark-issued agent JWT access tokens. Surfaces parsed claims with first-class types for `act`, `cnf`, and `authorization_details`.

## Public API
- `function verifyAgentToken(token: string, opts: VerifyOptions): Promise<AgentTokenClaims>`
- `function decodeAgentToken(token: string): AgentTokenClaims` — deprecated; no signature check
- Types: `AgentTokenClaims`, `ActorClaim`, `ConfirmationClaim`, `AuthorizationDetail`, `VerifyOptions`

## VerifyOptions
- `authUrl: string` — JWKS URL is `${authUrl}/.well-known/jwks.json`
- `expectedIssuer: string`
- `expectedAudience: string | string[]`
- `leeway?: number` — clock tolerance seconds (default 0)

## AgentTokenClaims fields
- Standard: `sub`, `aud`, `iss`, `exp`, `iat`
- `scope?: string`
- `act?: ActorClaim` — RFC 8693 actor (delegation chain)
- `cnf?: ConfirmationClaim` — `{ jkt }` for DPoP binding (RFC 9449)
- `authorization_details?: AuthorizationDetail[]` — RFC 9396 entries (each `{ type, ... }`)
- `raw: jose.JWTPayload` — full untyped payload

## Error mapping
- `JWTExpired` → `TokenError("token expired")`
- `JWSSignatureVerificationFailed` → `TokenError("invalid signature")`
- `JWTClaimValidationFailed` → `TokenError("claim validation failed: <claim> <code>")`
- Other → `TokenError("token verification failed: …")`
- `decodeAgentToken` malformed → `TokenError("malformed JWT: …")`

## Internal dependencies
- `jose` — `createRemoteJWKSet`, `jwtVerify`, `decodeJwt`
- `errors.ts` — `TokenError`

## Notes
- JWKS fetch is cached automatically by `jose.createRemoteJWKSet`.
- `decodeAgentToken` is exported for trusted edge-middleware use only.
- `verifyAgentToken` is NOT re-exported through `index.ts` (only `decodeAgentToken` and the claim types are).
