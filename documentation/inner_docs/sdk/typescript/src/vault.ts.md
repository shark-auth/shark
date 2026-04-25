# vault.ts

**Path:** `sdk/typescript/src/vault.ts`
**Type:** Token Vault client (3rd-party token retrieval)
**LOC:** 144

## Purpose
Fetches fresh, server-refreshed third-party OAuth credentials from the Shark Token Vault — agents trade a stored connection ID for a live provider token (Slack, Google, etc.) without ever holding the refresh token.

## Public API
- `class VaultClient`
  - `constructor(opts: VaultClientOptions)`
  - `exchange(referenceToken: string): Promise<VaultToken>` — GET `/admin/vault/connections/{id}/token`
- Types: `VaultToken`, `VaultClientOptions`

## VaultToken shape
- `accessToken: string`
- `expiresAt?: number` — Unix seconds
- `provider?: string`
- `scopes: string[]` — normalized from `scopes[]` or space-delimited `scope`

## Constructor options
- `authUrl: string`
- `accessToken: string` — calling agent's bearer (mutable internally on refresh)
- `onRefresh?: () => Promise<string>` — invoked on 401; returns a new agent token
- `maxRetries?: number` — default 2 — caps refresh-and-retry attempts
- `connectionsPath?: string` — default `/admin/vault/connections`

## Error mapping
- 401 + no callback / retries exhausted → `VaultError("agent not authorized", 401)`
- 403 → `VaultError("missing scope for vault access", 403)`
- 404 → `VaultError("connection not found: <id>", 404)`
- Other non-200 → `VaultError(message, status)`

## Internal dependencies
- `http.ts` — `httpRequest`
- `errors.ts` — `VaultError`

## Notes
- The agent never sees the provider's refresh token — vault keeps it server-side.
- `_accessToken` is mutated on successful refresh so subsequent calls use the new token.
- `referenceToken` is the connection ID, despite the parameter name.
