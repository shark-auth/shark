# dev_email.tsx

**Path:** `admin/src/components/dev_email.tsx`
**Type:** React component (page)
**LOC:** ~310
**Replaces:** `dev_inbox.tsx` (deleted)

## Purpose

Dev Email — captures all outgoing emails when `email.provider === "dev"`. Split list + right-side slide-over detail showing magic links, verification, and password-reset emails so the developer can click through them locally without configuring SMTP.

## Visibility Rule

**Critical:** This component is ONLY accessible when `adminConfig.email.provider === 'dev'`.

- `App.tsx` reads `adminConfig.email?.provider` from `GET /api/v1/admin/config` and passes it to `Sidebar` as `emailProvider`.
- `layout.tsx` Sidebar injects the `dev-email` nav entry into the OPERATIONS group only when `emailProvider === 'dev'`.
- `App.tsx` useEffect redirects any visit to `'dev-email'` page to `'overview'` if `emailProvider !== 'dev'` (once config loads).

## Exports

- `DevEmail` (named) — page component.
- Internal helpers: `EmailSlideOver`, `ClearConfirm`, `EmailTypeChip`, `detectEmailType`, `relativeTime`, `extractMagicLink`.

## Props / hooks

- props: `{}` (receives nothing; wires up internally)
- `useAPI('/admin/dev/emails')` — list + manual/auto refresh
- `usePageActions({ onRefresh: refresh })` — wires `r` keyboard shortcut
- State: `selected` (email row), `confirmClear`, `clearing`, `search`
- Auto-polls every 3 seconds via `setInterval(refresh, 3000)`

## API calls

- `GET /api/v1/admin/dev/emails` — list all captured emails (newest first)
- `GET /api/v1/admin/dev/emails/{id}` — fetch full email detail (html/text bodies)
- `DELETE /api/v1/admin/dev/emails` — clear all captured emails

**Note:** The brief expected `/admin/dev/email/inbox` but the actual backend uses `/admin/dev/emails` — this component uses the actual endpoints. The backend routes are registered in `internal/api/router.go` lines 664-666.

## Backend endpoint status

ALL endpoints exist and are wired:
- `GET /api/v1/admin/dev/emails` → `handleListDevEmails`
- `GET /api/v1/admin/dev/emails/{id}` → `handleGetDevEmail`
- `DELETE /api/v1/admin/dev/emails` → `handleDeleteAllDevEmails`

Source: `internal/api/devinbox_handlers.go`, `internal/api/router.go:664-666`.

## Composed by

- `App.tsx` route table: `'dev-email': DevEmail`
- `layout.tsx` Sidebar OPERATIONS group (conditional injection)

## Design constraints (.impeccable.md v3)

- Monochrome only — no hex literals, no color tints
- Hairline borders: `1px solid var(--hairline)` only — no box-shadows
- Square corners: `borderRadius: 3` max
- Dense rows: 7px row padding
- Right-side slide-over (`position: fixed; right: 0`) — NEVER a modal
- Inline confirm strip (no window.confirm, no centered modal) for destructive clear action
- CLIFooter at bottom: `shark dev email tail`

## Detail slide-over (EmailSlideOver)

Fixed panel `position:fixed; top:0; right:0; bottom:0; width:420px`. Three tabs: HTML (iframe sandboxed), Text (mono pre), Raw (source dump). Magic link auto-extracted and surfaced with open + copy actions. Dismissed by X button or Escape key.

## Conditional sidebar wiring pattern

In `layout.tsx`, the OPERATIONS section uses a `baseItems` override:

```tsx
const baseItems = section.group === 'OPERATIONS' && emailProvider === 'dev'
  ? [audit, webhooks, { id: 'dev-email', ... }, ...rest]
  : section.items;
```

This pattern (vs. a `devOnly` flag) was chosen because visibility depends on config value, not just dev_mode boolean. Future conditional nav entries should follow this same pattern.

## Notes

- 404 detection short-circuits to a "Dev mode required" explainer.
- Top warning banner reminds operator to switch provider before production.
- `detectEmailType` bins subjects into magic_link / verify / password_reset / invite.
- Magic link extraction supports both `href="..."` and bare URL patterns.
- `extractMagicLink` used in both list row and detail panel.
