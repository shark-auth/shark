# Contract — M2M rule flag

`RuleSpec.M2M` is a top-level boolean flag (not a `Require` kind) that locks a rule to agent-typed callers. When set, a rule that matches path + method + AppID AND whose `Require` predicate would otherwise pass still denies any caller whose `Identity.ActorType != identity.ActorTypeAgent`.

## Semantics

Evaluation order inside `Engine.Evaluate`:

1. Rule iteration — find the first rule matching path + method + AppID.
2. **M2M gate** — if `rule.M2M == true` and `id.ActorType != ActorTypeAgent`, return `Decision{ Allow: false, Kind: DecisionDenyForbidden, Reason: "rule requires agent (m2m)", MatchedRule: rule }`.
3. `Require` predicate — normal grammar evaluation (see `require_grammar.md`).
4. `Scopes` AND-combined.

The M2M gate fires **before** `Require` so the deny reason is unambiguous: an operator reading the audit log sees "rule requires agent (m2m)" rather than "role X required" or "scope Y required" that might also have failed.

## Actor types

```go
// internal/identity/identity.go
const (
    ActorTypeHuman ActorType = "human"
    ActorTypeAgent ActorType = "agent"
)
```

`ActorType` is populated by the request-time identity resolver — session cookie / JWT / API key. `ActorTypeAgent` is implied by `AuthMethodAPIKey` and by agent-issued JWTs. A request that resolved to an authenticated user via cookie or bearer JWT is `ActorTypeHuman`.

## Example RuleSpec JSON

```json
{
  "name": "webhooks-ingest-m2m-only",
  "pattern": "/webhooks/ingest/*",
  "methods": ["POST"],
  "require": "authenticated",
  "scopes": ["webhooks:write"],
  "m2m": true
}
```

This rule says: the caller must be authenticated (any agent or user), must hold the `webhooks:write` scope, AND must be agent-typed. A phished human session carrying `webhooks:write` still fails with `"rule requires agent (m2m)"`.

## Why M2M is a flag, not a Require predicate

Two reasons:

1. **Composition.** `m2m: true` composes with every `Require` string (`authenticated`, `role:X`, `scope:X`, `tier:X`, etc.). Making it a standalone `Require` kind would force operators to choose between "require scope X" and "require m2m"; they usually want both.
2. **Error clarity.** A pre-`Require` gate produces a distinct reason string that operators can filter on in audit logs. Folding it into `Require` would collapse that surface.

## Storage + wire

The flag lives on `proxy_rules.m2m INTEGER NOT NULL DEFAULT 0` (SQLite, 0/1 encoded). See `migration/00024_m2m.md`.

Wire format (admin CRUD + YAML import): `"m2m": true` / `m2m: true`. Omit for legacy rules — the default is `false` (human-inclusive), matching the pre-v1.5 behaviour.

## Audit

The proxy audit log field `actor_type` (added in Lane A, confirmed in Lane B6b) records the resolved identity kind on every decision. Searching `actor_type=human AND reason contains "m2m"` surfaces cases where a human attempted an M2M-gated route — useful as an early phishing / compromised-session signal.
