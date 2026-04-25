# consents_manage.tsx

**Path:** `admin/src/components/consents_manage.tsx`
**Type:** React component (page)
**LOC:** 323

## Purpose
Consents administration page — lists active OAuth consent grants with filter (all / has-expiry / no-expiry), scope filter, and free-text search across agent name / client_id. Per-row "Revoke" action opens a confirmation modal.

## Exports
- `Consents` (named) — page component.
- `RevokeConsentModal`, `FallbackEmptyState` (internal helpers).
- `splitScope`, `truncateMiddle`, `relativeTime` (utility helpers in-file).

## Props / hooks
- props: `{}`.
- State: `filter`, `scopeFilter`, `query`, `revokeModal`.
- `useAPI('/admin/oauth/consents')` for list + refresh.
- `usePageActions({ onRefresh })` to register the toolbar refresh keyboard shortcut.

## API calls
- GET `/api/v1/admin/oauth/consents`
- DELETE `/api/v1/admin/oauth/consents/{id}` (issued by RevokeConsentModal)

## Composed by
- `App.tsx` route table (consents page).

## Notes
- The legacy `/auth/consents` endpoint is session-authenticated — under admin-key auth it would 401. The page detects 401/403 errors and degrades into an explanatory empty state via `FallbackEmptyState`, keeping the dashboard usable while admin-wide listing is incomplete.
- CLI parity footer: `shark consent list` / `shark consents list --user <user-id>`.
- Scope chips derived by splitting whitespace-separated `c.scope` strings; distinct scopes power the filter dropdown.
