# Migration 00025 — branding.design_tokens

## SQL

```sql
-- +goose Up
-- PROXYV1_5 §4.11: design tokens live as a JSON blob so the dashboard
-- can evolve its token tree without schema migrations per field.
ALTER TABLE branding ADD COLUMN design_tokens TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE branding DROP COLUMN design_tokens;
```

## Purpose

Adds a JSON-encoded TEXT column on `branding` so the dashboard's design-tokens editor can persist a full color / typography / spacing / motion tree without requiring a schema migration per field. The shape is intentionally opaque to the server — the API handler validates only that the payload decodes as a JSON object.

## Column details

- Name: `design_tokens`
- Type: `TEXT NOT NULL DEFAULT ''`
- Default: empty string. Pre-existing branding rows inherit this on upgrade — design tokens are opt-in, not required.
- Encoding: JSON object, stringified. `null` is normalized to `{}` at the handler layer so the column never holds a literal `null`.

## Why a JSON blob, not per-field columns

Two reasons:

1. **Evolution speed.** A design-tokens tree like `{ colors.primary, colors.secondary, typography.scale.h1, ... }` has dozens of keys. Adding a column per key means a migration per design iteration; that's hostile to design velocity.
2. **Consumer tolerance.** The hosted-page renderer treats the tokens as opaque CSS custom properties. It doesn't need per-column lookup; a single JSON decode per request is fine.

The alternative — a schemaless key-value table — was rejected because the application always reads the full tree at once, so the JSON blob is strictly simpler without sacrificing performance.

## Consumers

### Writers

- `PATCH /api/v1/admin/branding/design-tokens` — overwrites the column with the JSON-encoded payload. See `api/admin_branding_design_tokens.md`.

### Readers

- `Store.GetBranding` + `Store.ResolveBranding` return the decoded map alongside the existing per-column fields.
- Hosted-page renderers (login, paywall) embed the tokens as CSS custom properties on the `:root` element so components can reference `var(--color-primary)` etc.

## Wire shape

Always a JSON object, even when empty:

```json
{ "design_tokens": {} }
```

The dashboard is free to enforce its own schema on the way in — the server only cares that the payload decodes cleanly.

## Related

- `api/admin_branding_design_tokens.md` — the admin API that writes this column.
- `api/paywall_route.md` — one of the hosted-page renderers that consumes it.
