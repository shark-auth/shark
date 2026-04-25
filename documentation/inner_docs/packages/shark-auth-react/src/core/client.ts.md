# client.ts

**Path:** `packages/shark-auth-react/src/core/client.ts`
**Type:** Auth-aware fetch client factory
**LOC:** 25

## Purpose
Creates a thin `fetch` wrapper that auto-attaches the SharkAuth publishable key and bearer access token to every request against the auth server.

## Public API
- `interface SharkClient { fetch(path: string, init?: RequestInit): Promise<Response> }`
- `createClient(authUrl: string, publishableKey: string): SharkClient`

The returned `fetch`:
1. Resolves `path` against the trimmed `authUrl` base.
2. Sets header `X-Shark-Publishable-Key: <publishableKey>`.
3. Reads access token from sessionStorage via `getAccessToken()`; if present, sets `Authorization: Bearer <token>`.
4. Forwards the call to global `fetch` with merged headers.

## Internal dependencies
- `getAccessToken` from `./storage`

## Used by (consumer-facing)
- `SharkProvider` — exposes the client through context as `ctx.client`.
- `MFAChallenge`, `PasskeyButton`, `OrganizationSwitcher`, `UserButton` — all call `ctx.client.fetch(...)` for backend interactions.
- `useAuth().signOut` indirectly (via provider) hits `/oauth/revoke` through this client.

## Notes
- No DPoP support here — DPoP proofs are added at the `getToken({ dpop, method, url })` layer in `SharkProvider`, not auto-attached by `createClient`.
- Headers are merged via `new Headers(init.headers)` so caller overrides survive.
- Trailing slash on `authUrl` is stripped to avoid `//path` joins.
