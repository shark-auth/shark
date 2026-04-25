# proxy_admin_v15_handlers.go

**Path:** `internal/api/proxy_admin_v15_handlers.go`
**Package:** `api`
**LOC:** 289
**Tests:** likely integration-tested

## Purpose
PROXYV1_5 Lane B admin handlers, kept in a dedicated file so the v1.5 surface stays auditable against `PROXYV1_5.md`. Three concerns: per-user billing tier flips, free-form design tokens for branding, and YAML import for proxy rules (legacy wire form bridge).

## Handlers exposed
- `handleSetUserTier` (line 34) — PATCH `/admin/users/{id}/tier`. Body `{tier: "free"|"pro"}`. Persists into `users.metadata`, audits `user.tier.set`, returns refreshed user + tier.
- `handleSetDesignTokens` (line 100) — PATCH `/admin/branding/design-tokens`. Body `{design_tokens: {...}}` — free-form JSON object so the dashboard can evolve `colors.*`, `typography.*`, `spacing.*`, `motion.*` without schema migrations. Audits `branding.design_tokens.set`.
- `handleImportYAMLRules` (line 172) — POST `/admin/proxy/rules/import`. Body `{yaml: "..."}`. Accepts either `{rules: [...]}` envelope or a bare list. Per-row failures are collected (not aborting); returns `{imported: N, errors: [...]}`.

## Key types
- `setUserTierRequest` (line 25) — `{tier}` (only `free`/`pro` recognized).
- `setDesignTokensRequest` (line 92) — `{design_tokens: map[string]any}`.
- `importYAMLRulesRequest` (line 141)
- `yamlRuleEnvelope` (line 149), `yamlRule` (line 153) — local mirror of the legacy `proxy.rules[]` YAML shape.

## Helpers
- `itoaFallback` (line 272) — local int-to-string for batch error reporting.

## Imports of note
- `gopkg.in/yaml.v3` — YAML parsing
- `internal/storage` — `SetUserTier`, `UpdateBranding`, `CreateProxyRule`, `GetUserByID`, `GetBranding`

## Wired by
- `internal/api/router.go:701` (set user tier), `:706` (design tokens), `:689` (yaml import)

## Notes
- Lane B §B5 is removing the legacy `config.ProxyConfig.Rules` field; this importer keeps the legacy YAML wire form alive.
- Tier is stored under `users.metadata` JSON (`{"tier": "..."}`); `tierFromMetadata` in `proxy_resolvers.go` is the parsed reader.
