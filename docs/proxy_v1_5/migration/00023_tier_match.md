# Migration 00023 — proxy_rules.tier_match

## SQL

```sql
-- +goose Up
ALTER TABLE proxy_rules ADD COLUMN tier_match TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE proxy_rules DROP COLUMN tier_match;
```

## Purpose

Adds a structured `tier_match` column to `proxy_rules` so dashboards can render "this rule is paywalled on tier X" without string-parsing the `require` field. The new column exists alongside `require: tier:<name>` — both compile to the same `ReqTier` predicate in the engine — but `tier_match` is the preferred column when a tool needs to *filter* rows by tier without evaluating them.

## Column details

- Name: `tier_match`
- Type: `TEXT NOT NULL DEFAULT ''`
- Default: empty string (= "no tier gate"). Pre-existing rows automatically inherit this default on upgrade.

## Consumers

### Writers

- `POST /api/v1/admin/proxy/rules/db` — accepts `tier_match` in the create body.
- `PATCH /api/v1/admin/proxy/rules/db/{id}` — accepts `tier_match` as an optional pointer field.
- `POST /api/v1/admin/proxy/rules/import` — YAML import recognises a top-level `tier_match:` key.

### Readers

- `Store.ListProxyRules` returns the column via the shared `proxy_rules_sqlite.go` select list.
- `refreshProxyEngineFromDB` reads the value and compiles it into a `ReqTier` requirement when non-empty. Today that wiring goes through `require: tier:<name>` at compile time; the structured column is projected into JSON responses for the dashboard's benefit.

## Wire shape

JSON on rule responses:

```json
{ "tier_match": "pro" }
```

Empty string is omitted (`omitempty`) in response payloads but explicitly accepted on write to let admins clear the tier gate.

## Related

- `require: tier:<name>` — see `contracts/require_grammar.md`.
- Paywall routing — see `contracts/decision_kinds.md` (DecisionPaywallRedirect).
- User tier admin — see `api/admin_users_tier.md`.
