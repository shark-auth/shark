# Dev Email Inbox

## Overview

The Dev Email surface captures all outgoing mail when `email.provider` is set to `dev`. It is only rendered when the admin config confirms dev provider is active. It is never shown in production mode.

## Backend Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/admin/dev/emails` | List all captured emails |
| `GET` | `/api/v1/admin/dev/emails/{id}` | Fetch a single email with full body |
| `DELETE` | `/api/v1/admin/dev/emails` | Clear all captured emails |

All endpoints require admin authentication (401 without credentials). If the server is not in dev mode, endpoints return 404 and the UI shows a "not dev mode" placeholder instead of the inbox.

## Envelope Shape

`GET /api/v1/admin/dev/emails` returns one of the following shapes (all accepted by the frontend):

```json
{ "emails": [ ...items ] }
{ "data":   [ ...items ] }
{ "items":  [ ...items ] }
[ ...items ]
```

Each item must include:

| Field | Aliases | Required by |
|-------|---------|-------------|
| `id` | — | row key, detail fetch |
| `to` | `to_addr`, `recipient` | banner "last received" display |
| `subject` | — | table, type detection |
| `created_at` | `received_at`, `timestamp` | banner sort, relative time |
| `html_body` | `html`, `body_html`, `body` | slide-over preview |

## Banner State Contract

The dev-mode banner derives its state entirely from the latest backend poll — never from stale local state. Implementation (`admin/src/components/dev_email.tsx`):

1. `useAPI('/admin/dev/emails')` polls every 3 seconds via `setInterval(refresh, 3000)`.
2. `allEmails` is derived from `data` on every poll response.
3. `lastDelivered` is a `useMemo` that sorts `allEmails` descending by timestamp and returns the most recent item.
4. The banner renders:
   - **No emails yet:** "Dev email capture active. Switch provider before production."
   - **Email(s) present:** "Dev capture active — last received `<to>` `<relative time>`. Switch provider before production."

This means the banner always reflects the delivery state visible in the inbox table — it cannot show "received" when nothing has been received, and it updates automatically as new mail arrives.

## UI Surface

- **Table:** sortable by received time, filterable by to/subject.
- **Slide-over:** right-side fixed panel (420 px), never a modal. Shows HTML preview, plain text, raw JSON tabs plus magic-link copy button when detected.
- **Clear:** inline confirm strip (no modal). DELETE clears server-side, then refreshes.
- **Keyboard:** `r` to refresh, `Esc` to close slide-over.

## CLI Equivalent

```sh
shark dev email tail
```

## Activation

Switch provider in **Settings → Email Delivery** or via CLI:

```sh
shark mode dev
```

To revert to live sending:

```sh
shark mode prod
```
