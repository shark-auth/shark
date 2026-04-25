# context.ts

**Path:** `packages/shark-auth-react/src/hooks/context.ts`
**Type:** React context definition + auth value types
**LOC:** 30

## Purpose
Declares the singleton `AuthContext` that `SharkProvider` populates and every hook/component reads from. Defines the option/result shapes for `getToken`.

## Public API (exported types + the context)
- `interface GetTokenOptions { dpop?: boolean; method?: string; url?: string }`
- `interface GetTokenResult { token: string; dpop?: string }`
- `interface AuthContextValue {`
  - `isLoaded: boolean`
  - `isAuthenticated: boolean`
  - `user: User | null`
  - `session: Session | null`
  - `organization: Organization | null`
  - `client: ReturnType<typeof createClient>` — bound `SharkClient`
  - `getToken: (opts?: GetTokenOptions) => Promise<string | GetTokenResult | null>`
  - `signOut: () => Promise<void>`
  - `authUrl?: string` / `publishableKey?: string` — internal use by `SignIn`/`SignUp`
- `}`
- `const AuthContext = createContext<AuthContextValue | null>(null)`

## Internal dependencies
- `react` (`createContext`)
- `core/types.{User, Session, Organization}`
- `core/client.createClient` (type only)

## Used by (consumer-facing)
- All hooks (`useAuth`, `useUser`, `useSession`, `useOrganization`) consume via `useContext(AuthContext)`.
- All components needing `client` (`MFAChallenge`, `PasskeyButton`, `OrganizationSwitcher`, `SharkCallback`, `SignIn`, `SignUp`) consume directly via `React.useContext(AuthContext)`.
- Re-exported from the package root, so power users can subscribe directly with `useContext(AuthContext)`.

## Notes
- Default value is `null` — every hook throws if used outside `<SharkProvider>`.
- `getToken` returns `string` for the bare-token path, `GetTokenResult` (with DPoP proof) when `opts.dpop === true`, or `null` if no session is recoverable.
