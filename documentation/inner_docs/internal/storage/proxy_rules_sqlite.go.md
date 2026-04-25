# proxy_rules_sqlite.go

**Path:** `internal/storage/proxy_rules_sqlite.go`
**Package:** `storage`
**LOC:** 215
**Tests:** `proxy_rules_sqlite_test.go`

## Purpose
SQLite implementation of proxy-rule CRUD. Writes `tier_match` + `m2m` alongside the legacy columns so every Lane A/B v1.5 enhancement round-trips via the same path the dashboard and YAML import use.

## Interface methods implemented
- `CreateProxyRule` (14) — JSON-encodes `Methods` + `Scopes`
- `GetProxyRuleByID` (44)
- `ListProxyRules` (51) — ordered `priority DESC, created_at ASC`
- `ListProxyRulesByAppID` (71)
- `UpdateProxyRule`, `DeleteProxyRule`
- Internal scanners (`scanProxyRuleRow`, `scanProxyRuleRows`)

## Tables touched
- proxy_rules

## Imports of note
- `database/sql`, `encoding/json`, `time`

## Used by
- `internal/api/proxy_rules.go` admin CRUD handlers
- `internal/proxy/engine.go` for rule reload + evaluation
- `internal/proxy/loader.go` for YAML + DB merge

## Notes
- `proxyRuleSelectCols` constant (line 41) centralizes the SELECT column list so single-row, multi-row, and app-scoped queries scan the same ordered columns. Adding/reordering columns means one place to change.
- Sort order matches the engine's evaluation order: highest priority first, ties broken by earliest creation.
- `tier_match` empty string = "no tier constraint"; non-empty = strict equality with Identity.Tier.
- `m2m` stored as INT (0/1) via `boolToInt` per the package convention.
