# authentication.tsx

**Path:** `admin/src/components/authentication.tsx`
**Type:** React component (page)
**LOC:** 450

## Purpose
Read-only Authentication settings dashboard — surfaces the live auth-method configuration (password policy, OAuth providers, magic links, passkeys, JWT mode) plus rendered email-template previews.

## Exports
- `Authentication` (named) — page component.
- `AuthCell` (helper, internal) — cell renderer for the read-only grid.

## Props / hooks
- props: `{}`.
- `useAPI('/admin/config')` for `config`, `loading`, `error`.
- `React.useState` for `previewTpl`, `previewHTML`, `previewErr`.
- `openPreview(tpl)` issues `API.get('/admin/email-preview/' + tpl)`.

## API calls
- GET `/api/v1/admin/config`
- GET `/api/v1/admin/email-preview/{tpl}`

## Composed by
- `App.tsx` route table (`page === 'authentication'`).

## Notes
- `// @ts-nocheck` — predates the TS migration.
- Display-only: any edits live elsewhere (Settings page handles writes; Branding > Email handles template editing).
- OAuth providers list is hard-coded to Google / GitHub / Apple / Discord and matched against `cfg.social_providers || cfg.oauth_providers`.
- Renders three top sections (Password Policy, OAuth Providers, two-column row of Magic Links + Passkeys) followed by JWT and email-template preview blocks (lower in file).
- Trailing `<CLIFooter>` reinforces CLI parity per design system.
