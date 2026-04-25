# sqlite_webhooks.go

**Path:** `internal/storage/sqlite_webhooks.go`
**Package:** `storage`
**LOC:** 232
**Tests:** `sqlite_webhooks_test.go`

## Purpose
SQLite implementation of webhook + webhook-delivery methods on `Store`. Backs the outbound event dispatcher in `internal/webhooks`.

## Interface methods implemented
- Webhooks: `CreateWebhook` (13), `GetWebhookByID` (25), `ListWebhooks` (31), `ListEnabledWebhooksByEvent` (46) (uses `LIKE` on JSON-encoded events array with `"name"` quoting to avoid prefix false-positives), `UpdateWebhook` (61), `DeleteWebhook` (70)
- Webhook deliveries: `CreateWebhookDelivery` (103), `UpdateWebhookDelivery` (116), `GetWebhookDeliveryByID` (128), `ListWebhookDeliveriesByWebhookID` (137) (keyset cursor pagination, `created_at|id`), `ListPendingWebhookDeliveries` (180) (drives the retry scheduler), `DeleteWebhookDeliveriesBefore`

## Tables touched
- webhooks
- webhook_deliveries

## Imports of note
- `database/sql`
- `strings` — for cursor splitting + WHERE building
- `time` — RFC3339 UTC for `next_retry_at` filter

## Used by
- `internal/webhooks/dispatcher.go` for queueing + retry
- `internal/api/webhooks.go` admin handlers

## Notes
- Events column is a JSON array stored as TEXT. `ListEnabledWebhooksByEvent` uses a `LIKE %"event"%` pattern; the surrounding quotes prevent matching `user.created_v2` when searching `user.created`. If the install ever grows to thousands of webhooks, switch to a `webhook_events` join table.
- Defaults: `Events` empty string → `"[]"` (line 14).
- Cursor format documented inline (line 137): `created_at|id` for stable keyset pagination ordered DESC.
