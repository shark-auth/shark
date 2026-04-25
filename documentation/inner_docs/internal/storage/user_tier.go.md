# user_tier.go

**Path:** `internal/storage/user_tier.go`
**Package:** `storage`
**LOC:** 82
**Tests:** `user_tier_test.go` (indirectly).

## Purpose
Helpers for reading + writing a user's billing tier (`free`/`pro`) without forking the schema. Tier is stored inside `users.metadata` JSON under the `"tier"` key — same bucket already used for app-defined per-user fields. Lane A's Claims baker and these helpers share one source of truth without a schema migration. (PROXYV1_5 §4.10.)

## Interface methods implemented
- `SetUserTier(ctx, userID, tier)` (23) — reads metadata, deserializes (resets to `{}` on corruption rather than erroring), writes back `tier=<value>`, updates `updated_at`. Returns `sql.ErrNoRows` if the user is missing.
- `GetUserTier(ctx, userID)` (63) — returns the `"tier"` string from metadata; returns `""` (without error) when the user exists but has no tier set; returns `sql.ErrNoRows` only when the user itself is missing.

## Tables touched
- users (read + UPDATE on `metadata` and `updated_at`)

## Imports of note
- `database/sql`, `encoding/json`, `time`

## Used by
- `internal/api/users.go` — admin tier toggle (H3 dashboard)
- `internal/identity` — Claims baker reads `tier` so JWTs/session claims carry it for the proxy engine
- `internal/proxy/engine.go` — tier-gated rule evaluation

## Notes
- Round-tripping through metadata preserves every other key (custom app data, etc.).
- Corruption-tolerant: malformed JSON in `metadata` is treated as "no tier" rather than fatal — keeps an admin from being locked out of recovering a row.
- Edge default: callers treat empty string as `"free"` at the rule-evaluation boundary.
