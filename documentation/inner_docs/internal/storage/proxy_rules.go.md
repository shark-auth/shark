# proxy_rules.go

**Path:** `internal/storage/proxy_rules.go`
**Package:** `storage`
**LOC:** 39
**Tests:** indirectly via `proxy_rules_sqlite_test.go`.

## Purpose
Type-only file: declares `ProxyRule`, the DB-backed override row for the reverse-proxy rule engine (Phase 6.6 / Wave D). YAML rules from `sharkauth.yaml` remain the bootstrap source of truth; rows here layer on top so admins can author + edit rules from the dashboard without restarting the server.

## Type
- `ProxyRule` (line 14):
  - `Pattern` — chi-style path pattern (e.g. `/api/orgs/{id}`, `/v1/public/*`)
  - `Methods` — empty slice for "any", otherwise uppercased HTTP verbs
  - `Require` / `Allow` — exactly one is set; engine compiles via existing `proxy.parseRequirement`
  - `Scopes` — required OAuth scopes
  - `TierMatch` (PROXYV1_5 §4.10) — constrains rule to callers whose `Identity.Tier` equals this value; mismatch returns `DecisionPaywallRedirect` so the proxy can route browsers to an upgrade page
  - `M2M` (PROXYV1_5 §4.17) — when true, requires agent-typed identity; humans denied with "rule requires agent (m2m)"
  - `Priority` — higher evaluated first

## Used by
- `internal/storage/proxy_rules_sqlite.go` — implementation.
- `internal/proxy/engine.go` — rule compilation + evaluation.
- `internal/api/proxy_rules.go` admin CRUD handlers.

## Notes
- Migration 00015 added base table; 00022 added `app_id`; 00023 added `tier_match`; 00024 added `m2m`.
- The struct is intentionally flat — no nested objects — so JSON round-trip via the dashboard is trivial.
