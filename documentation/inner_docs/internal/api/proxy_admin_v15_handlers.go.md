# proxy_admin_v15_handlers.go

**Path:** `internal/api/proxy_admin_v15_handlers.go`
**Package:** `api`
**LOC:** 289
**Tests:** likely integration-tested

## Purpose
PROXYV1_5 Lane B admin handlers, kept in a dedicated file so the v1.5 surface stays auditable against `PROXYV1_5.md`. Two concerns remain post-Phase H: per-user billing tier flips and free-form design tokens for branding. YAML import handler removed.

## Handlers exposed
- `handleSetUserTier` — PATCH `/admin/users/{id}/tier`. Body `{tier: "free"|"pro"}`. Persists into `users.metadata`, audits `user.tier.set`, returns refreshed user + tier.
- `handleSetDesignTokens` — PATCH `/admin/branding/design-tokens`. Body `{design_tokens: {...}}` — free-form JSON object so the dashboard can evolve `colors.*`, `typography.*`, `spacing.*`, `motion.*` without schema migrations. Audits `branding.design_tokens.set`.
- ~~`handleImportYAMLRules`~~ — **DELETED Phase H.** POST `/admin/proxy/rules/import` route removed. Use DB-backed rules CRUD instead.

## Key types
- `setUserTierRequest` — `{tier}` (only `free`/`pro` recognized).
- `setDesignTokensRequest` — `{design_tokens: map[string]any}`.
- ~~`importYAMLRulesRequest`, `yamlRuleEnvelope`, `yamlRule`~~ — **DELETED Phase H.**

## Helpers
- `itoaFallback` — local int-to-string for batch error reporting.

## Imports of note
- `internal/storage` — `SetUserTier`, `UpdateBranding`, `GetUserByID`, `GetBranding`
- ~~`gopkg.in/yaml.v3`~~ — **REMOVED Phase H.**

## Wired by
- `internal/api/router.go` (set user tier), (design tokens)

## Notes
- YAML import route (`/admin/proxy/rules/import`) and handler deleted in Phase H; OpenAPI spec updated accordingly.
- Tier is stored under `users.metadata` JSON (`{"tier": "..."}`); `tierFromMetadata` in `proxy_resolvers.go` is the parsed reader.
