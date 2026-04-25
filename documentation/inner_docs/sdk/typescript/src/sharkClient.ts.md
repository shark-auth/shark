# sharkClient.ts

**Path:** `sdk/typescript/src/sharkClient.ts`
**Type:** Top-level SDK client
**LOC:** 123

## Purpose
Top-level entrypoint that bundles a DPoP-aware `fetch` wrapper with all v1.5 admin namespaces, so consumers construct one client and reach everything through it.

## Public API
- `class SharkClient`
  - `constructor(opts: SharkClientOptions)` — wires up admin namespaces from `baseUrl` and `adminKey` (falls back to `accessToken` if `adminKey` omitted).
  - `.proxyRules` — `ProxyRulesClient`
  - `.proxyLifecycle` — `ProxyLifecycleClient`
  - `.branding` — `BrandingClient`
  - `.paywall` — `PaywallClient` (no auth)
  - `.users` — `UsersClient`
  - `.agents` — `AgentsClient`
  - `.fetch(input, init?)` — DPoP-signed (or Bearer) wrapper around native `fetch`
- `interface SharkClientOptions`

## Constructor options
- `accessToken: string` — required, the calling agent's bearer token
- `dpopProver?: DPoPProver` — when present, every `.fetch()` call is signed with `Authorization: DPoP <token>` + `DPoP: <proof>`; otherwise plain `Bearer`
- `userAgent?: string` — defaults to `@sharkauth/node/0.1.0`
- `adminKey?: string` — used by all admin namespaces; falls back to `accessToken`
- `baseUrl?: string` — required for any admin namespace or paywall use

## Internal dependencies
- `dpop.ts` — proof signing
- `proxyRules.ts`, `proxyLifecycle.ts`, `branding.ts`, `paywall.ts`, `users.ts`, `agents.ts` — admin namespaces

## Behavior
- `.fetch()` strips query/fragment from the URL via DPoP layer, sets `User-Agent` and `Accept: application/json`, then dispatches to native fetch.
- No retries at this layer — admin namespaces handle their own error mapping; `httpRequest` (used by namespaces) handles timeouts.

## Notes
- Admin namespaces are created eagerly in the constructor — there's no lazy init.
- This client does not handle token refresh; that is the caller's responsibility (or `VaultClient`'s `onRefresh`).
