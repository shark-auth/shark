# magicLink.ts

**Path:** `sdk/typescript/src/magicLink.ts`

Magic link send client for the TypeScript SDK. Public endpoint — no admin key needed.

## Exports

- `MagicLinkClient` — class
- `MagicLinkClientOptions` — interface (`{ baseUrl: string }`)
- `SendMagicLinkResult` — interface (`{ message: string }`)

## Constructor

```ts
new MagicLinkClient({ baseUrl: string })
```

## Methods

### `sendMagicLink(email, redirectUri?) → Promise<SendMagicLinkResult>`

POSTs JSON `{ email[, redirect_uri] }` to `POST /api/v1/auth/magic-link/send`. Server applies per-email rate limit (1/60 s). Always returns 200 (anti-enumeration). Throws `SharkAPIError` on non-200.

`redirectUri` must be on the server's `allowed_callback_urls` allowlist.

## Route

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/v1/auth/magic-link/send` | None |

## Notes

- 503 `feature_disabled` when email provider not configured.
- Exported from `index.ts`.
