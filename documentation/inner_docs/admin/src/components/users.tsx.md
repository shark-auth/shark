# users.tsx

**Path:** `admin/src/components/users.tsx`
**Type:** React component (page)
**LOC:** 700+

## Purpose
Users management page—paginated table with search, filters (verified|MFA|auth method), bulk actions, slide-over detail view.

## Exports
- `Users()` (default) — function component

## Features
- **Search** — debounced (300ms) search by name, email, or ID
- **Filters** — verified (yes|no), MFA (on|off), auth method (password|oauth|passkey|magic_link)
- **Pagination** — 25 per page, reset to page 1 on filter change
- **Table columns** — user avatar+email, verified badge, MFA status, auth method, orgs, roles, created, last active, actions
- **Bulk selection** — checkboxes, select-all header
- **Bulk actions** — export CSV, assign role (deferred), add to org (deferred), delete (CLI-only)
- **Detail slide-over** — user info, email/ID copy, password reset, delete, impersonate
- **Deep linking** — `/admin/users/<id>` selects user
- **Create flow** — `?new=1` opens create modal, stripped after

## Hooks used
- `useAPI('/users?...')` — paginated, searchable users list
- `useURLParam('search', ...)` — filter state in URL
- `useToast()` — success/error feedback
- `usePageActions({ onRefresh, onNew })` — keyboard shortcuts (r=refresh, n=new)

## State
- `selected` — currently viewing user
- `creating` — show create modal
- `query` — search text
- `filterVerified`, `filterMfa`, `filterMethod` — filter selections
- `debouncedQuery` — debounced search (after 300ms)
- `checked` — Set of selected user IDs for bulk actions
- `page`, `perPage` — pagination

## API calls
- `GET /api/v1/users?page=...&per_page=...&search=...&email_verified=...&mfa_enabled=...&auth_method=...` — list users
- `DELETE /api/v1/users/{id}` — delete user

## Composed by
- App.tsx

## Notes
- Uses `// @ts-nocheck`
- Empty state shows "No users found"
- URL params sync with component state
- CSV export done client-side from already-loaded list
