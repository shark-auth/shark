# webhook_emit.go

**Path:** `internal/api/webhook_emit.go`
**Package:** `api`
**LOC:** 37
**Tests:** none colocated (used by every handler that emits)

## Purpose
Tiny utility module for safe webhook emission from any handler. Wraps the optional `WebhookDispatcher` so emission sites don't repeat the nil check, and provides a `publicUser` projection that strips sensitive fields from outbound payloads.

## Functions
- `(*Server).emit` (line 13) — nil-safe wrapper around `WebhookDispatcher.Emit`. Errors are logged via `slog.Error` but never returned — webhook delivery is always best-effort async.
- `userPublic` (line 32) — converts `*storage.User` → `publicUser` for outbound webhook payloads.

## Key types
- `publicUser` (line 24) — webhook-safe user projection: `{id, email, email_verified, name?, created_at}`. Excludes password hash and MFA secret.

## Imports of note
- `internal/storage` — `User`, webhook event constants (`storage.WebhookEvent*`)

## Wired by
- Called by signup, admin user-create, organization-create/delete, MFA enable, etc. — anywhere a webhook event is emitted.

## Notes
- Failed webhook emissions never fail the originating request — the contract is fire-and-forget.
- New event payloads should pass through a similar projection to keep secrets out of webhook bodies.
