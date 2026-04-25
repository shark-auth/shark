# SharkProvider.tsx

**Path:** `packages/shark-auth-react/src/components/SharkProvider.tsx`
**Type:** React context provider — root of the SDK
**LOC:** 175

## Purpose
The required wrapper. Hydrates auth state from the persisted access token, exposes `client`, `getToken`, `signOut`, and the resolved `user`/`session`/`organization` via `AuthContext`. Wires up optional DPoP.

## Public API
- `interface SharkProviderProps { publishableKey: string; authUrl: string; dpop?: boolean; children: React.ReactNode }`
- `function SharkProvider(props): JSX.Element`

Injects an `AuthContextValue`:
```ts
{ isLoaded, isAuthenticated, user, session, organization,
  client: SharkClient, getToken, signOut,
  authUrl, publishableKey }
```

## Required setup
```tsx
<SharkProvider
  publishableKey="pk_live_..."
  authUrl="https://auth.example.com"
  dpop /* optional, enables sender-constrained tokens */
>
  <App />
</SharkProvider>
```

Plus a route at `/shark/callback` rendering `<SharkCallback />` (callback URI is hard-coded in `core/auth.ts`).

## How it composes
1. **Mount:** creates a memoized `SharkClient` from `(authUrl, publishableKey)`.
2. **DPoP:** if `dpop` prop set, calls `generateDPoPProver()` once and stashes the prover.
3. **Hydrate effect:** reads access token from storage; if present, fetches `GET /api/v1/users/me`; on success populates `{ user, session, organization }`. On network error, falls back to `decodeClaims(token)` to derive a minimal user from JWT claims (`sub`, `email`, `firstName`, `lastName`, `imageUrl`, `jti`, `exp`). On decode failure, `clearAll()` and resets state.
4. **getToken(opts?):** returns the stored access token; auto-refreshes via `exchangeToken({ refreshToken, dpopProof? })` if expired; when `opts.dpop` requested, requires `method`+`url` and returns `{ token, dpop }`. On refresh failure, clears state and returns `null`.
5. **signOut():** best-effort `POST /oauth/revoke` (form-encoded `token=...`), then `clearAll()` and resets state.

## Internal dependencies
- `core/client.createClient`
- `core/storage.{getAccessToken,getRefreshToken,clearAll}`
- `core/jwt.decodeClaims`
- `core/auth.exchangeToken`
- `core/dpop.{generateDPoPProver, DPoPProver}`
- `hooks/context.{AuthContext, GetTokenOptions, GetTokenResult}`
- `core/types.{User, Session, Organization}`

## Used by (consumer-facing)
- Required ancestor for every other component (`SignIn`, `SignUp`, `UserButton`, `MFAChallenge`, `PasskeyButton`, `OrganizationSwitcher`, `SharkCallback`) and every hook (`useAuth`, `useUser`, `useSession`, `useOrganization`).

## Notes
- Hydration `<API>/api/v1/users/me` response shape: `{ user, session, organization }` — all optional.
- On signOut, revoke is best-effort; failures don't block local clear.
- File contains a stray `)` on line 86 (after `getToken` callback closes) — likely a known build-time anomaly worth verifying.
