# proxy_rules_handlers.go

**Path:** `internal/api/proxy_rules_handlers.go`
**Package:** `api`
**LOC:** 530
**Tests:** likely integration-tested

## Purpose
DB-backed proxy rule CRUD (PROXYV1_5 Lane A). These override-rules layer on top of YAML-defined rules at engine refresh time. Validates payloads through the same compile path the engine uses, so any row that survives create/update is guaranteed to load cleanly on `refreshProxyEngineFromDB`.

## Handlers exposed
- `handleListProxyRules` (line 103) — GET `/admin/proxy/rules/db`. Optional `?app_id=` filter. `{data, total}`.
- `handleCreateProxyRule` (line 131) — POST. Validates require/allow grammar; if caller supplies an `id` that already exists, returns 200 (idempotent) when payloads match, else 409 conflict.
- `handleGetProxyRule` (line 247) — GET `/{id}`.
- `handleUpdateProxyRule` (line 263) — PATCH. Pointer-typed fields for partial update.
- `handleDeleteProxyRule` (line 362) — DELETE.

## Internal entry point
- `refreshProxyEngineFromDB` (line 411) — reloads engine ruleset after any mutation; called by lifecycle reload as well.

## Key types
- `proxyRuleResponse` (line 23) — full DB-row mirror (id, app_id, name, pattern, methods, require, allow, scopes, enabled, priority, tier_match, m2m, timestamps).
- `createProxyRuleRequest` (line 68), `updateProxyRuleRequest` (line 83)

## Helpers
- `proxyRuleToResponse` (line 40), `proxyRuleMatchesCreate` (line 218), `stringSlicesEqual` (line 234)
- `validateProxyRulePayload` (line 445), `validRequireString` (line 482), `normalizeMethods` (line 504), `newProxyRuleID` (line 524)

## Imports of note
- `internal/proxy` — compile-path validation
- `internal/storage` — `ProxyRule` CRUD (`ListProxyRules`, `ListProxyRulesByAppID`, `CreateProxyRule`, `GetProxyRuleByID`, `UpdateProxyRule`, `DeleteProxyRule`)

## Wired by
- `internal/api/router.go:684-688`

## Notes
- Available regardless of proxy enable state so admins can stage rules before flipping the proxy on.
- `tier_match` + `m2m` are v1.5 Lane B fields; `tier_match=""` means "all tiers".
- Idempotent create with explicit `id` lets external systems push the same rule repeatedly without 409.
