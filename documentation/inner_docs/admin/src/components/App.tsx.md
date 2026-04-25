# App.tsx

**Path:** `admin/src/components/App.tsx`
**Type:** React component (page shell)
**LOC:** 450+

## Purpose
Root app container—handles auth state, page routing, sidebar + topbar layout, keyboard shortcuts, command palette, development mode, onboarding flow.

## Exports
- `App` (default, function component) — main entry point

## Props
None (uses localStorage for state persistence)

## State / hooks
- `apiKey` (localStorage: `shark_admin_key`) — admin API key
- `page` — current page (overview|users|sessions|orgs|agents|...|compliance|flow|get-started)
- `tweaks` — UI preferences (sidebarCollapsed, font, showPreview)
- `editMode` — edit mode flag (responds to postMessage)
- `paletteOpen` — command palette visibility
- `cheatsheetOpen` — keyboard shortcuts visibility
- `walkthroughOpen` — onboarding modal visibility
- `adminConfig` — fetched admin config (dev_mode, etc.)

## Page routing map
Maps page strings to component exports:
- overview → `Overview`
- users, sessions, orgs, agents, devices → CRUD pages
- audit, webhooks, keys → operations
- settings, rbac, sso, auth, proxy → configuration
- dev-inbox, explorer, debug, schemas → developer tools
- compliance, migrations, branding, flow → enterprise features

## API calls
- `GET /api/v1/admin/config` — check dev mode
- `GET /api/v1/admin/stats` — redirect logic for onboarding

## Composed by
- Components: Sidebar, TopBar, Overview, Users, Sessions, Organizations, etc.
- Overlays: KeyboardCheatsheet, CommandPalette, HelpButton, Walkthrough
- Providers: ToastProvider

## Features
- **Auth state** — login/logout via `onLogin` callback
- **Deep linking** — URL drives page state
- **Bootstrap detection** — redirects new orgs to get-started flow
- **Dev mode gating** — some features only in dev mode
- **Phase gating** — some features gated to future phases (showPreview override)
- **Keyboard shortcuts** — Cmd/Ctrl+K opens command palette
- **Tweaks panel** — sidebar collapse, font switching (Manrope|Inter|IBM|Space)
- **Edit mode** — responds to parent iframe postMessage for design editing

## Notes
- Uses `// @ts-nocheck` to skip type checking
- Stores onboarding state in localStorage (`shark_admin_onboarded`, `shark_walkthrough_seen`)
- Emits `shark-auth-expired` event on 401 to clear auth
