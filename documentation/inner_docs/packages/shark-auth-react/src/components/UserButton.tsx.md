# UserButton.tsx

**Path:** `packages/shark-auth-react/src/components/UserButton.tsx`
**Type:** React component — avatar dropdown
**LOC:** 160

## Purpose
Renders a circular avatar (image or initials fallback) that opens a small inline-styled dropdown with profile/account links and a sign-out action.

## Public API
- `interface UserButtonProps { profileUrl?: string; manageAccountUrl?: string; afterSignOutUrl?: string }`
- `function UserButton(props): JSX.Element | null`

### Props
- `profileUrl` — anchor target for the "Profile" menu item. Default `'/profile'`.
- `manageAccountUrl` — anchor target for "Manage account". Default `'/account'`.
- `afterSignOutUrl` — `window.location.href` destination after `signOut()`. Default `'/'`.

## How it composes
1. Pulls `{ isLoaded, isAuthenticated, signOut }` from `useAuth()` and `{ user }` from `useUser()`.
2. Returns `null` if not loaded, not authenticated, or no user — never renders for signed-out users.
3. Computes initials: first letter of `firstName` + `lastName` (uppercased), falling back to `email[0]`.
4. Toggles a dropdown via local `open` state. Closes on outside-click via a `mousedown` listener attached to `document` while open.
5. Avatar is keyboard-accessible: `role="button"`, `tabIndex={0}`, opens on Enter/Space.
6. Sign-out flow: closes menu, awaits `signOut()`, then navigates to `afterSignOutUrl`.

## Internal dependencies
- `hooks/useAuth`
- `hooks/useUser`

## Used by (consumer-facing)
- Header/navbar integration:
  ```tsx
  <SignedIn><UserButton afterSignOutUrl="/" /></SignedIn>
  ```

## Notes
- All styles are inline `React.CSSProperties` — no CSS dependency, no theming hook. Override visuals by wrapping or by forking the component.
- Dropdown z-index is hard-coded to `9999` to defeat most layout layers.
- `aria-expanded` reflects open state; menu items have `role="menuitem"`.
