# webhooks.tsx

**Path:** `admin/src/components/webhooks.tsx`
**Type:** React component (page)
**LOC:** ~500

## Purpose
Webhook event subscription management—define hooks on user, session, auth events; test delivery; retry logs; delivery status.

## Exports
- `Webhooks()` (default) — function component

## Features
- **Webhook list** — endpoint, events subscribed, status (healthy|failing), last delivery
- **Create flow** — URL, event selection (user.created, user.updated, session.created, auth.failed, etc.)
- **Detail pane** — delivery history, test webhook, retry failed deliveries
- **Event types** — user, session, authentication, admin events
- **Retry mechanism** — exponential backoff, manual retry
- **Signature verification** — HMAC-SHA256 for security

## Hooks used
- `useAPI('/admin/webhooks')` — list webhooks
- `useAPI('/admin/webhooks/{id}/deliveries')` — delivery history

## State
- `selected` — current webhook
- `creating` — create modal
- `testingId` — webhook being tested
- `filter` — by status (all|healthy|failing)

## API calls
- `GET /api/v1/admin/webhooks` — list
- `POST /api/v1/admin/webhooks` — create
- `PATCH /api/v1/admin/webhooks/{id}` — update
- `DELETE /api/v1/admin/webhooks/{id}` — delete
- `POST /api/v1/admin/webhooks/{id}/test` — test delivery
- `GET /api/v1/admin/webhooks/{id}/deliveries` — delivery log
- `POST /api/v1/admin/webhooks/{id}/deliveries/{delivery_id}/retry` — retry failed

## Composed by
- App.tsx

## Notes
- Event payload includes actor (who triggered), timestamp, data
- Failed deliveries auto-retry with backoff
- Webhook signature (X-Shark-Signature) for security verification
