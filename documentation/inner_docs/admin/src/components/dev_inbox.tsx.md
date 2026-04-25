# dev_inbox.tsx

**Path:** `admin/src/components/dev_inbox.tsx`
**Type:** React component (page)
**LOC:** 368

## Purpose
Dev Inbox — captures all outgoing emails when `server.dev_mode = true`. Split-pane list + detail view of magic links, verification, and password-reset emails so the developer can click through them locally without configuring SMTP.

## Exports
- `DevInbox` (named) — page component.
- Internal helpers: `EmailDetail`, `EmailTypeChip`, `detectEmailType`, `relativeTime`.

## Props / hooks
- props: `{}`.
- `useAPI('/admin/dev/emails')` for list + manual refresh.
- State: `selected` (email row), `clearing`.
- `useToast()`.

## API calls
- GET `/api/v1/admin/dev/emails`
- DELETE `/api/v1/admin/dev/emails` (clear all)

## Composed by
- `App.tsx` route table (dev-inbox page).

## Notes
- 404 detection short-circuits to a "Dev mode required" explainer so production admins don't see a broken page.
- Top warning banner reminds the operator that the inbox captures all outgoing mail and to switch providers before production.
- Subject heuristic (`detectEmailType`) bins the email type when the backend doesn't supply one.
- CLI parity: `shark dev inbox tail`.
