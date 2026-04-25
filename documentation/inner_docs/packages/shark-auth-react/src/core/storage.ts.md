# storage.ts

**Path:** `packages/shark-auth-react/src/core/storage.ts`
**Type:** sessionStorage-backed token store
**LOC:** 76

## Purpose
Wraps `sessionStorage` with SSR-safe getters/setters for the five SharkAuth-specific keys: access token, expiry, refresh token, PKCE code verifier, and post-login redirect URL.

## Public API
- `getAccessToken(): string | null` — returns null if missing **or** if `Date.now() > expiresAt` (caller is expected to refresh).
- `getRefreshToken(): string | null`
- `setAccessToken(token, expiresAt, refreshToken?)` — writes all three.
- `clearAll(): void` — removes every namespaced key.
- `getCodeVerifier(): string | null` / `setCodeVerifier(verifier)` — PKCE verifier persisted across the OAuth redirect.
- `getRedirectAfter(): string | null` / `setRedirectAfter(url)` — where to land after callback.

## Storage keys (all sessionStorage)
- `shark_access_token`
- `shark_access_token_expires_at`
- `shark_refresh_token`
- `shark_code_verifier`
- `shark_redirect_after`

## Internal dependencies
None — only the platform `window`/`sessionStorage` globals.

## Used by (consumer-facing)
- `core/client.ts` — pulls bearer token for every request.
- `core/auth.ts` — sets verifier+redirect at `startAuthFlow`, persists tokens after `exchangeToken`.
- `components/SharkProvider.tsx` — reads access/refresh on hydrate, calls `clearAll` on signOut/refresh-failure.
- `components/SharkCallback.tsx` — reads `shark_redirect_after` to decide where to send the user post-exchange.

## Notes
- `isBrowser` guard makes every helper a no-op under SSR (Next.js, RSC).
- All writes are wrapped in try/catch — Safari private mode and storage quota errors are swallowed silently.
- **Security tradeoff:** sessionStorage is XSS-readable. Refresh tokens here mean an XSS → silent session theft. The DPoP option mitigates token replay but not extraction.
