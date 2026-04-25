# webhook_handlers.go

**Path:** `internal/api/webhook_handlers.go`
**Package:** `api`
**LOC:** 417
**Tests:** likely integration-tested

## Purpose
Webhook subscription CRUD + delivery inspection + replay. Subscriptions are HMAC-signed (HMAC secret returned exactly once on create). Subscription event names are validated against a single registry of `KnownWebhookEvents` so typos surface on create, not silently never-fire.

## Handlers exposed
- `handleCreateWebhook` (line 71) — POST `/admin/webhooks`. Returns the secret once. Validates URL (https + non-loopback in prod) and event names.
- `handleListWebhookEvents` (line 117) — GET `/admin/webhooks/events`. Returns the canonical event registry as a sorted list (lets the dashboard populate a picker without hardcoding).
- `handleListWebhooks` (line 127) — GET. `{webhooks: [...]}`.
- `handleGetWebhook` (line 140) — GET `/{id}`.
- `handleUpdateWebhook` (line 161) — PATCH. URL/Events/Enabled/Description.
- `handleDeleteWebhook` (line 208) — DELETE.
- `handleTestWebhook` (line 227) — POST `/{id}/test`. Fires a `webhook.test` event so the operator can verify their endpoint.
- `handleListDeliveries` (line 279) — GET `/{id}/deliveries`. With `clampDeliveryLimit`.
- `handleReplayWebhookDelivery` (line 310) — POST `/{id}/deliveries/{deliveryId}/replay`.

## Key types
- `webhookResponse` (line 39), `webhookResponseWithSecret` (line 51, only at create)
- `createWebhookRequest` (line 65), `updateWebhookRequest` (line 154)

## Package state
- `KnownWebhookEvents` (line 25) — event registry: user.created/updated/deleted, session.created/revoked, mfa.enabled, org.created/deleted, org.member.added, system.audit_log, webhook.test.

## Helpers
- `webhookToResponse` (line 56), `validateWebhookURL` (line 372), `validateEvents` (line 389)
- `newWebhookSecret` (line 401), `clampDeliveryLimit` (line 409)

## Imports of note
- `internal/storage` — `Webhook`, `WebhookEvent*` constants
- `crypto/rand`, `encoding/hex` — secret generation
- `gonanoid` — id suffix (`wh_*`)

## Wired by
- `internal/api/router.go:495-504`

## Notes
- HMAC secret returned once on create; subsequent reads omit it.
- Replay re-uses the same delivery semantics as a live emit (signing, retries, etc.).
