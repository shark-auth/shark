# get_started.tsx

**Path:** `admin/src/components/get_started.tsx`
**Type:** React component (page)
**LOC:** ~400

## Purpose
Onboarding/setup wizard—guides new orgs through initial configuration (proxy, auth methods, first user, branding).

## Exports
- `GetStarted()` (default) — function component

## Features
- **Step 1: Proxy** — configure reverse proxy to protect upstream app
- **Step 2: Auth** — select auth methods (password, OAuth, passkey, magic link)
- **Step 3: Users** — add first user or import
- **Step 4: Branding** — customize login page logo/colors
- **Step 5: Deploy** — verify and go live
- **Progress** — step indicator, next/skip/back buttons
- **Skip option** — can skip to dashboard anytime

## Triggers
- Shown when `users.total === 0` and `shark_admin_onboarded` not set
- After completion, sets `shark_admin_onboarded = '1'` in localStorage

## Composed by
- App.tsx

## Notes
- Reduces time-to-value for new orgs
- Each step is independent; can be done in any order
- Guided setup reduces configuration errors
