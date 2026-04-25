# proxyRules.ts

**Path:** `sdk/typescript/src/proxyRules.ts`
**Type:** Admin namespace — DB-backed proxy rules CRUD
**LOC:** 204

## Purpose
Admin-key-authenticated CRUD for the embedded reverse-proxy's database-backed rule table. Used by the dashboard and operators to add/update/disable/import rules without restarting.

## Public API
- `class ProxyRulesClient`
  - `constructor(opts: ProxyRulesClientOptions)`
  - `listRules(opts?: { appId? }): Promise<ProxyRuleListResult>` — GET `/api/v1/admin/proxy/rules/db`
  - `createRule(spec: CreateProxyRuleInput): Promise<ProxyRuleMutationResult>` — POST, expects 201
  - `getRule(id: string): Promise<ProxyRule>` — GET `/db/{id}`
  - `updateRule(id, patch): Promise<ProxyRuleMutationResult>` — PATCH
  - `deleteRule(id): Promise<void>` — DELETE, expects 204
  - `importRulesYaml(yaml: string): Promise<ImportResult>` — POST `/api/v1/admin/proxy/rules/import`

## ProxyRule fields
`id, app_id, name, pattern, methods[], require, allow, scopes[], enabled, priority, tier_match, m2m, created_at, updated_at`

## Mutation envelope
`ProxyRuleMutationResult = { data: ProxyRule, engine_refresh_error?: string }` — the engine_refresh_error is non-empty when the DB write succeeded but the live engine refresh failed (caller should retry `/reload`).

## Import contract
`ImportResult = { imported: number, errors: ImportRuleError[] }` — partial success is the contract; per-row failures don't abort.

## Constructor options
- `baseUrl: string`
- `adminKey: string` — Bearer token

## Error mapping
- All non-success statuses → `SharkAPIError(message, code, status)` parsed from `{error:{code,message}}`.

## Internal dependencies
- `http.ts`, `errors.ts`

## Notes
- chi-style path patterns; `pattern` and `name` required on create.
- `tier_match` enables the paywall flow when set (e.g. `"pro"`).
- `m2m` denies non-agent (human) callers when true.
