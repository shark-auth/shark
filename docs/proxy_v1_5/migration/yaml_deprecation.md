# Migration — YAML proxy.rules deprecation

## Why

Pre-v1.5 SharkAuth loaded proxy rules from a `proxy.rules:` block in `sharkauth.yaml`. That coupled proxy configuration to process restarts: an operator who wanted to add, change, or disable a rule had to edit YAML and restart `shark serve`, which broke hot-path admin flows and made multi-instance deployments hard to keep in sync.

v1.5 moves the source of truth to the DB (`proxy_rules` table), managed via the admin API (`POST /api/v1/admin/proxy/rules/db` and friends). Rules live through restarts, replicate to every instance via the shared SQLite file or replicated backend, and can be mutated via CRUD + hot-reload without touching the process.

## Timeline

| Release | Behaviour |
|---|---|
| v1.5 (current) | `proxy.rules:` YAML is **ignored** at load time (the struct field was removed). `shark serve` emits a stderr WARNING on startup when the on-disk config still carries the block, pointing at this doc and at the import endpoint. `shark proxy` standalone command prints a deprecation stub and exits 2. |
| v2 (planned) | Presence of `proxy.rules:` is a **fatal startup error** — the warning graduates to a config-load failure so operators cannot silently ship the wrong rules. |

## What the warning looks like

```
WARNING: proxy.rules YAML section is deprecated in v1.5; move rules to the DB via POST /api/v1/admin/proxy/rules. See docs/proxy_v1_5/migration/yaml_deprecation.md
```

Emitted to stderr on every `shark serve` boot where the on-disk YAML carries a top-level `proxy:` block with a direct-child `rules:` key. Detection is a line-level scan (indent-aware) — nested `rules:` keys under unrelated blocks (e.g. `rbac.roles[].rules:`) do not trigger the warning.

## Migrating existing rules

### Option A — Bulk import (recommended)

Paste the legacy YAML block into the dashboard's "Import rules" pane (see `api/admin_proxy_rules_import.md`) or POST it directly:

```bash
cat <<'YAML' | jq -Rs '{ yaml: . }' \
  | curl -XPOST \
      -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
      -H "Content-Type: application/json" \
      --data-binary @- \
      https://auth.example.com/api/v1/admin/proxy/rules/import
rules:
  - path: /admin/*
    require: role:admin
  - path: /api/writes/*
    require: authenticated
    scopes: [webhooks:write]
  - path: /m2m/*
    require: authenticated
    m2m: true
YAML
```

Response:

```json
{ "imported": 3, "errors": [] }
```

### Option B — One rule at a time

Translate each legacy rule into a CRUD `POST`:

```bash
curl -XPOST \
  -H "Authorization: Bearer $SHARK_ADMIN_KEY" \
  -H "Content-Type: application/json" \
  https://auth.example.com/api/v1/admin/proxy/rules/db \
  -d '{
    "name":    "admin-only",
    "pattern": "/admin/*",
    "require": "role:admin"
  }'
```

Useful when you're partway through a migration and want to diff each rule against the UI before committing.

### Option C — CLI (planned for Lane E)

`shark proxy rules import --file rules.yaml` will wrap the bulk-import endpoint. Until Lane E lands, use the curl snippets above.

## Verifying the migration

1. Run the bulk import (above).
2. Inspect the count: `GET /api/v1/admin/proxy/rules/db | jq '.total'` should equal the number of rules you imported.
3. Remove the `proxy.rules:` block from `sharkauth.yaml`.
4. Restart `shark serve` — the WARNING should no longer appear.
5. Tail the proxy audit log and issue a test request against each migrated pattern to confirm the rule is live.

## Rolling back (unlikely, but documented)

The `ProxyRule` struct name is retained with deprecation godoc so any third-party code that imports it keeps compiling. The YAML loader no longer parses the field, so the only way to restore the legacy path is to cherry-pick the pre-Lane-B config.go and router.go changes — which is an explicit fork, not a config switch.

If you hit a migration blocker, keep the rules in the DB and disable individual rows by PATCHing `{"enabled": false}`. That's the reversible escape hatch; a full rollback is not.
