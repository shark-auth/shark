# jwt.ts

**Path:** `packages/shark-auth-react/src/core/jwt.ts`
**Type:** JWT decode + verify helpers
**LOC:** 35

## Purpose
Two-tier JWT helpers: a fast unverified browser-side claim decoder (used during hydration when offline) and a remote-JWKS verifier wrapping `jose`.

## Public API
- `interface SharkClaims extends JWTPayload { sub?, email?, firstName?, lastName?, orgId?, [key: string]: unknown }`
- `decodeClaims(token: string): SharkClaims` — splits on `.`, base64url-decodes the payload, JSON parses. Throws `Invalid JWT format` or `Failed to decode JWT payload`.
- `verifyToken(token: string, jwksUrl: string): Promise<JWTPayload>` — uses `createRemoteJWKSet(new URL(jwksUrl))` + `jwtVerify`. Returns the verified payload.

## Internal dependencies
- `jose` — `jwtVerify`, `createRemoteJWKSet`, `JWTPayload` type

## Used by (consumer-facing)
- `SharkProvider.hydrate()` falls back to `decodeClaims(token)` when `/api/v1/users/me` errors, so the user object survives offline/dev.
- `verifyToken` is exported but not called inside the package — it exists for SSR/middleware consumers that want to validate the token they received.

## Notes
- `decodeClaims` performs **no signature verification** — never trust `decodeClaims` output server-side; it's a UI-only convenience for extracting `email`, `firstName` etc. from a token already considered authentic by the auth server.
- `base64urlDecode` pads to a multiple of 4 and translates `-_` → `+/` before `atob`.
