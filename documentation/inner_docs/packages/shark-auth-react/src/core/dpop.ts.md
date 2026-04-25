# dpop.ts

**Path:** `packages/shark-auth-react/src/core/dpop.ts`
**Type:** RFC 9449 DPoP proof generator (browser WebCrypto)
**LOC:** 86

## Purpose
Generates a per-tab ECDSA P-256 keypair and produces fresh DPoP proof JWTs binding each request to that key, so a stolen access token cannot be replayed without the matching private key.

## Public API
- `interface DPoPProver { jkt: string; createProof(method: string, url: string, accessToken?: string): Promise<string> }`
- `generateDPoPProver(): Promise<DPoPProver>`
  - Creates an extractable ECDSA P-256 keypair via `crypto.subtle.generateKey`.
  - Computes `jkt` = base64url(SHA-256(canonical JWK)) per RFC 7638 (only `crv`,`kty`,`x`,`y` fields, sorted JSON).
  - Returns the prover.
- `createProof(method, url, accessToken?)`:
  - Builds JWT header `{ typ: 'dpop+jwt', alg: 'ES256', jwk: { kty, crv, x, y } }`.
  - Builds payload `{ jti: 16-byte random, htm: METHOD, htu: url-without-query-or-fragment, iat: now }`.
  - When `accessToken` is provided, adds `ath = base64url(SHA-256(accessToken))` (RFC 9449 §4.2).
  - Signs `header.payload` with ECDSA-SHA256, returns the compact JWS.

## Internal dependencies
- Browser globals: `crypto.subtle`, `crypto.getRandomValues`, `TextEncoder`, `btoa`.

## Used by (consumer-facing)
- `SharkProvider` lazily creates one prover when `dpop` prop is true (`generateDPoPProver().then(setProver)`), reuses it for every `getToken({ dpop, method, url })` call, and also feeds proofs to `exchangeToken` during silent refresh.

## How a consumer plugs it in
```tsx
<SharkProvider authUrl="…" publishableKey="…" dpop>
  <App />
</SharkProvider>
```

```ts
const { getToken } = useAuth()
const result = await getToken({ dpop: true, method: 'GET', url: 'https://api.example.com/me' })
// result is { token, dpop } — caller sets:
//   Authorization: DPoP <token>
//   DPoP: <result.dpop>
```

## Notes
- Keys are non-persistent (in-memory keypair object) — losing the tab loses the prover. Refresh fetches a new one. This is intentional for bearer-binding scope.
- `htu` must strip query+fragment to match server-side normalization.
