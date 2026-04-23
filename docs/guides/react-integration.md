# React Integration Guide — `@shark-auth/react`

Drop-in React components + hooks for SharkAuth. Works with Next.js App Router, Vite, CRA.

## Install

```bash
npm install @shark-auth/react
# or
pnpm add @shark-auth/react
```

Peer deps: React 18+, React DOM 18+.

## Register your app

1. Open the Shark admin dashboard → Applications → New.
2. Set **Integration mode** = `components`.
3. Add callback URL: `http://localhost:3000/shark/callback` (and production equivalent).
4. Copy the **publishable key** (`pk_live_...` or `pk_test_...`).

## Configure env vars

```
NEXT_PUBLIC_SHARK_URL=https://auth.yourdomain.com   # or http://localhost:8080
NEXT_PUBLIC_SHARK_KEY=pk_test_xxxxxxxx
```

For non-Next frameworks, use whatever env convention your bundler supports. The SDK reads nothing from `process.env` itself — you pass the values into `<SharkProvider>`.

## Wrap your app

### Next.js App Router

`app/providers.tsx`:

```tsx
'use client'
import { SharkProvider } from '@shark-auth/react'

export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <SharkProvider
      publishableKey={process.env.NEXT_PUBLIC_SHARK_KEY!}
      authUrl={process.env.NEXT_PUBLIC_SHARK_URL!}
    >
      {children}
    </SharkProvider>
  )
}
```

`app/layout.tsx`:

```tsx
import Providers from './providers'
export default function Root({ children }: { children: React.ReactNode }) {
  return <html><body><Providers>{children}</Providers></body></html>
}
```

Also add `transpilePackages: ['@shark-auth/react']` to `next.config.mjs`.

### Vite / CRA

```tsx
import { SharkProvider } from '@shark-auth/react'

createRoot(document.getElementById('root')!).render(
  <SharkProvider publishableKey={import.meta.env.VITE_SHARK_KEY} authUrl={import.meta.env.VITE_SHARK_URL}>
    <App />
  </SharkProvider>
)
```

## Drop in components

```tsx
import { SignIn, SignedIn, SignedOut, UserButton } from '@shark-auth/react'

<SignedOut>
  <SignIn redirectUrl="/dashboard" />
</SignedOut>
<SignedIn>
  <UserButton />
</SignedIn>
```

## Callback page

Create a route at `/shark/callback` that renders `<SharkCallback />`:

### Next.js

```tsx
// app/shark/callback/page.tsx
'use client'
import { SharkCallback } from '@shark-auth/react'
export default function Callback() { return <SharkCallback /> }
```

### React Router

```tsx
<Route path="/shark/callback" element={<SharkCallback />} />
```

`SharkCallback` reads `?code=`, exchanges for tokens via PKCE, and redirects to the URL stored in `sessionStorage` (set by `<SignIn redirectUrl="..." />`).

## Hooks

```tsx
import { useAuth, useUser, useSession, useOrganization } from '@shark-auth/react'

function Profile() {
  const { isLoaded, isAuthenticated, signOut } = useAuth()
  const { user } = useUser()

  if (!isLoaded) return null
  if (!isAuthenticated) return <a href="/">Sign in</a>
  return (
    <div>
      <p>{user?.email}</p>
      <button onClick={signOut}>Sign out</button>
    </div>
  )
}
```

All hooks throw if called outside `<SharkProvider>`.

## MFA / Passkeys

```tsx
import { MFAChallenge, PasskeyButton } from '@shark-auth/react'

<MFAChallenge onSuccess={() => router.push('/dashboard')} />
<PasskeyButton mode="signin" onSuccess={() => router.refresh()} />
```

## Organizations

```tsx
import { OrganizationSwitcher, useOrganization } from '@shark-auth/react'

<OrganizationSwitcher />
const { organization } = useOrganization()
```

## Troubleshooting

**Session lost on tab close.** Tokens are kept in `sessionStorage` by design — security over convenience. Switch to `localStorage` by forking `core/storage.ts` if you need persistence.

**"useAuth must be used within SharkProvider".** The hook was called outside the provider tree. Check that every consumer is nested under `<SharkProvider>`.

**`window is not defined` during Next.js build.** Either mark the route `'use client'` or add `export const dynamic = 'force-dynamic'`. The SDK guards SSR access but `<SharkProvider>` must render in the client to hydrate.

**Callback loop.** Verify the callback URL in your app registration exactly matches the URL React navigates to (`window.location.origin + '/shark/callback'`).

## Example app

Full working example at `examples/react-next/` in the shark-auth repo.
