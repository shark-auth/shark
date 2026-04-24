# Admin API — Proxy rules YAML import

## Purpose

Bulk-imports a YAML document describing a rule list into the `proxy_rules` table. Every row runs through the same validation the CRUD create path uses, so a YAML rule that survives import is guaranteed to compile cleanly on the next engine refresh. On per-row failure the handler collects the error and continues so a single typo doesn't abort the whole batch.

The primary use case is the v1.5 YAML-deprecation migration (see `migration/yaml_deprecation.md`): operators paste their old `proxy.rules:` block into the dashboard import pane, the server writes every rule into the DB, and the live engine is refreshed before the response returns.

## Route

| Method | Path | Handler symbol |
|---|---|---|
| POST | `/api/v1/admin/proxy/rules/import` | `Server.handleImportYAMLRules` |

## Auth required

Admin API key.

## Request shape

```json
{
  "yaml": "rules:\n  - path: /admin/*\n    require: role:admin\n  - path: /m2m/*\n    require: authenticated\n    m2m: true\n"
}
```

The YAML payload accepts two shapes:

1. Top-level `rules:` envelope — matches the legacy `proxy.rules` block 1:1.
2. Bare list — a YAML sequence of rule objects with no outer key.

Per-rule schema (`yamlRule` in `proxy_admin_v15_handlers.go`):

```yaml
- app_id: app_abc        # optional
  name: block-writes     # optional; defaults to path when empty
  path: /api/writes/*    # required
  methods: [POST, PUT]   # optional
  require: authenticated # optional — see contracts/require_grammar.md
  allow: ""              # optional — only "anonymous"
  scopes: [foo:write]    # optional
  enabled: true          # optional; defaults to true
  priority: 100          # optional
  tier_match: pro        # optional
  m2m: false             # optional
```

## Response shape

### Success (200)

```json
{
  "imported": 2,
  "errors": [
    { "index": "1", "name": "bad-rule", "error": "pattern must start with '/'" }
  ]
}
```

`imported` is the count of rows that made it into the DB. `errors` is always a non-nil array (empty `[]` on full success) so clients can render "N failed" without null-checks.

### Error

```json
{ "error": { "code": "invalid_yaml", "message": "yaml: line 3: ..." } }
```

Top-level errors: `invalid_request` (bad outer JSON or missing `yaml` field), `invalid_yaml` (parse failure). Per-row errors land inside the `errors` array and never fail the batch.

## Status codes

- `200 OK` — handler reached a decision; inspect `imported` + `errors` to see what landed.
- `400 Bad Request` — bad outer JSON or YAML parse failure.
- `401 Unauthorized` — missing/invalid admin key.

## Side effects

- DB writes: one `INSERT` into `proxy_rules` per validated rule. Failures are collected into `errors` without rolling back prior INSERTs in the batch — partial success is the contract.
- Engine refresh: `refreshProxyEngineFromDB` is invoked after all rows are processed so a successful import is live on the next request.
- Audit log: one `proxy.rules.imported` entry per request (batch-level, not per row) with `ActorType=admin`.

## Frontend hint

The dashboard's YAML-import pane should be a drag-and-drop zone plus a textarea fallback. Submit as JSON `{ "yaml": <file-contents> }`. After the response returns, render a two-column table of `errors` (index + name + message) and surface `imported` as a success toast. Because partial success is possible, disable the "delete original YAML" button until `errors.length === 0`. Pair the form with a linkout to `migration/yaml_deprecation.md` so first-time migrators can self-serve.
