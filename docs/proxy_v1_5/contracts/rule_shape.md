# Contract — Rule shape

The canonical rule shape accepted by `POST /api/v1/admin/proxy/rules/db` and `POST /api/v1/admin/proxy/rules/import`. Every field has a documented default, so omissions never produce undefined behaviour.

## Wire schema (JSON)

```json
{
  "id":         "string (UUID-shaped; server-generated on create, required on update)",
  "app_id":     "string (optional; empty = global rule)",
  "name":       "string (required; human label, shown in dashboards + audit logs)",
  "pattern":    "string (required; chi-style path — see below)",
  "methods":    ["string"],
  "require":    "string (see require_grammar.md)",
  "allow":      "string (only \"anonymous\" accepted; see require_grammar.md)",
  "scopes":     ["string"],
  "enabled":    true,
  "priority":   0,
  "tier_match": "string (free | pro | empty)",
  "m2m":        false,
  "created_at": "RFC3339 UTC (read-only; server-stamped on create)",
  "updated_at": "RFC3339 UTC (read-only; server-stamped on every write)"
}
```

## Field reference

### `id` (string)
Server-generated on create via `newProxyRuleID()` — UUID-shaped string with a `rule_` prefix. Stable across updates; use it in PATCH/DELETE path params.

### `app_id` (string, optional)
Scopes the rule to a single application's proxy domain. Empty string = global rule that applies to every app's catch-all listener. App-scoped rules are loaded by the `AppResolver` only when a request matches that app's proxy domain; they do not contribute to the global engine refresh.

### `name` (string, required)
Human-readable label. Surfaces in dashboards and audit log entries. `validateProxyRulePayload` rejects an empty name.

### `pattern` (string, required)
Chi-style path pattern. Must start with `/`. Supported syntax:

- Exact: `/api/foo`
- Trailing prefix: `/api/foo/*` matches `/api/foo` plus everything under it (zero or more extra segments)
- Single-segment wildcard: `/api/*/deep` matches any second segment
- `{id}` placeholder: `/api/{id}` is treated identically to `/api/*` (MVP does not capture the value; rules engine only needs match/no-match)
- `/*` alone matches every path

### `methods` ([]string, optional)
HTTP method filter. Case-insensitive — `"get"` in YAML still matches `GET`. Empty array = any method. Duplicates are deduped at compile time.

### `require` / `allow` (string, exclusive)
Exactly one must be set. See `require_grammar.md` for every accepted value.

### `scopes` ([]string, optional)
AND-combined extra scopes. Every listed scope must be present in `Identity.Scopes`; the first missing one is surfaced in the deny reason. A missing scope on an authenticated caller returns `DecisionDenyForbidden` (403), not anonymous.

### `enabled` (bool, default true)
When false, the rule is persisted but excluded from engine refresh. Use this to stage rules before enabling, or to deactivate a rule without deleting it (preserving priority + audit history).

### `priority` (int, default 0)
Sort order. The engine evaluates rules in ascending slice index; storage returns them in `Priority DESC, created_at ASC` order so higher priority = first match wins. DB rows default to 0 unless explicitly bumped.

### `tier_match` (string, optional)
Added in migration 00023. When non-empty, constrains the rule to callers whose `Identity.Tier` equals this value. Functionally overlaps with `require: tier:<name>`; `tier_match` is the structured column used by some dashboards that want to render "this rule is paywalled" without parsing the `require` string. Both compile to the same `ReqTier` at engine time.

### `m2m` (bool, default false)
Added in migration 00024. When true, restricts the rule to agent-typed callers. See `m2m_rule_flag.md` for full semantics.

### `created_at` / `updated_at` (RFC3339 UTC)
Server-managed. Read-only on update requests — any incoming value is ignored.

## Compile-time validation

Every rule that survives `validateProxyRulePayload` on the wire layer compiles cleanly in `proxy.compileRule`. The two validators are intentionally paired — the dashboard gets synchronous, accurate feedback on save, and the engine never sees a row it can't compile.

Validation errors produce HTTP 400 with `error.code = invalid_proxy_rule` and the specific message (empty name, bad pattern prefix, unknown require string, etc.).

## SDK sketch

TypeScript:

```ts
export interface ProxyRule {
  id:         string;
  app_id:     string;
  name:       string;
  pattern:    string;
  methods:    string[];
  require:    string;  // see require_grammar.md
  allow:      string;  // "" or "anonymous"
  scopes:     string[];
  enabled:    boolean;
  priority:   number;
  tier_match: string;
  m2m:        boolean;
  created_at: string;  // ISO-8601
  updated_at: string;
}
```

Python:

```python
from typing import TypedDict

class ProxyRule(TypedDict):
    id: str
    app_id: str
    name: str
    pattern: str
    methods: list[str]
    require: str
    allow: str
    scopes: list[str]
    enabled: bool
    priority: int
    tier_match: str
    m2m: bool
    created_at: str
    updated_at: str
```
