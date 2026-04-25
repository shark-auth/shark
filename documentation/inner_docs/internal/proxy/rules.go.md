# rules.go

**Path:** `internal/proxy/rules.go`
**Package:** `proxy`
**LOC:** 697
**Tests:** `rules_test.go`

## Purpose
Defines the rule matching engine: path/method/requirement compilation and first-match-wins evaluation for reverse proxy request flow.

## Key types / functions
- `RuleSpec` (struct, line 18) — raw rule shape (AppID, path, methods, require, scopes, M2M flag)
- `RequirementKind` (enum, line 36) — predicate family (Anonymous, Authenticated, Role, Permission, Agent, Scope, Tier, GlobalRole)
- `Requirement` (struct, line 102) — compiled predicate (Kind + Value)
- `Rule` (struct, line 131) — compiled rule ready for matching (path pattern, methods map, requirement, M2M)
- `Engine` (struct, line 224) — atomic.Pointer to rule slice, first-match-wins evaluator
- `Decision` (struct, line 208) — evaluation outcome (Allow, MatchedRule, Reason, Kind, PaywallApp, RequiredTier)
- `DecisionKind` (enum, line 179) — outcome classification (Allow, DenyAnonymous, DenyForbidden, PaywallRedirect)
- `Engine.Evaluate()` (line 309) — main entry point: finds first matching rule and returns Decision
- `Engine.SetRules()` (line 254) — atomically replaces rule set (called by admin CRUD)
- `evaluatePrimary()` (line 394) — dispatches on requirement kind to run predicate

## Imports of note
- `internal/identity` — Identity type for requirement evaluation

## Wired by
- `internal/api/proxy_rules_handlers.go` (rule CRUD endpoints)
- `internal/proxy/listener.go` (NewListener compiles rules into Engine)
- `internal/proxy/proxy.go` (ReverseProxy.director calls engine.Evaluate)

## Used by
- Reverse proxy request flow (`:8080` and per-listener ports) when matching inbound requests

## Notes
- Rules stored behind atomic.Pointer for lock-free reads on hot path (line 225).
- Path matching supports literals, wildcards (`/api/*`), and `{id}` placeholders (line 589).
- M2M flag evaluated before requirement to surface clear "rule requires agent" reason even when other predicates fail (line 325).
- ReqPermission placeholder exists; Phase 6.5 will wire RBAC store; currently always denies (line 418).
- Tier mismatches trigger PaywallRedirect instead of generic 403 to enable upgrade UX (line 451).
