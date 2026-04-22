# @shark-auth/react

React components + hooks for SharkAuth — drop-in auth for Next.js, Vite, CRA.

## Install

```bash
npm install @shark-auth/react
```

Peer: React 18+.

## Quick start

```tsx
import { SharkProvider, SignIn, SignedIn, SignedOut, UserButton } from '@shark-auth/react'

<SharkProvider publishableKey={process.env.NEXT_PUBLIC_SHARK_KEY!} authUrl={process.env.NEXT_PUBLIC_SHARK_URL!}>
  <SignedOut><SignIn redirectUrl="/dashboard" /></SignedOut>
  <SignedIn><UserButton /></SignedIn>
</SharkProvider>
```

Add a `/shark/callback` route rendering `<SharkCallback />`.

## Components

`SharkProvider`, `SignIn`, `SignUp`, `SignedIn`, `SignedOut`, `UserButton`, `MFAChallenge`, `PasskeyButton`, `OrganizationSwitcher`, `SharkCallback`.

## Hooks

`useAuth`, `useUser`, `useSession`, `useOrganization`.

## Docs

Full guide: [docs/guides/react-integration.md](https://github.com/shark-auth/shark/blob/main/docs/guides/react-integration.md)

## License

MIT
