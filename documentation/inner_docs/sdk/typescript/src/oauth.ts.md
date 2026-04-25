# oauth.ts

**Path:** `sdk/typescript/src/oauth.ts`

RFC 7009 token revocation and RFC 7662 token introspection client for the TypeScript SDK.

## Exports

- `OAuthClient` — class
- `OAuthClientOptions` — interface
- `IntrospectResult` — interface
- `TokenTypeHint` — type alias (`"access_token" | "refresh_token"`)

## Constructor options

```ts
new OAuthClient({ baseUrl: string, adminKey?: string })
```

`adminKey` is an admin API key (`sk_live_...`). Required for introspection; optional for revocation.

## Methods

### `revokeToken(token, tokenTypeHint?) → Promise<void>`

Posts form-encoded body to `POST /oauth/revoke`. Server always returns 200. Throws `SharkAPIError` on non-200.

### `introspectToken(token) → Promise<IntrospectResult>`

Posts form-encoded body to `POST /oauth/introspect`. Returns `{ active: boolean, scope?, client_id?, sub?, exp?, iss?, token_type?, ...claims }`. Active=false for invalid/expired tokens.

## Routes

| Method | Path | Auth |
|--------|------|------|
| POST | `/oauth/revoke` | None required (Bearer forwarded if set) |
| POST | `/oauth/introspect` | Bearer admin key |

## Notes

- 503 `feature_disabled` when OAuth AS not configured.
- Revocation is idempotent — always safe to call.
- Exported from `index.ts`.
