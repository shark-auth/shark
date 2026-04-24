# Proxy v1.5 docs

This is the authoritative reference for the SharkAuth reverse-proxy + hosted-page surface introduced in v1.5. It targets three audiences in decreasing order of traffic: dashboard authors wiring the admin UI, SDK authors projecting endpoints into TypeScript + Python, and operators migrating from the legacy YAML-driven proxy.

Every file in this tree follows the same skeleton — title, route/signature, auth requirements, request + response shapes, status codes, side effects, and a frontend-wiring hint. Contract docs describe the shared primitives (decision kinds, rule shape, require grammar) that multiple endpoints reference; migration docs explain the schema deltas introduced in migrations 00023–00025 plus the YAML-deprecation timeline.

## Layout

```
docs/proxy_v1_5/
├── README.md                      — this file
├── api/                           — HTTP surface for the v1.5 proxy admin
│   ├── admin_proxy_rules_db.md          CRUD for DB-backed override rules
│   ├── admin_proxy_lifecycle.md         start/stop/reload/status for the embedded proxy
│   ├── admin_proxy_rules_import.md      YAML bulk-import of legacy rules
│   ├── admin_users_tier.md              per-user tier mutator (feeds paywall)
│   ├── admin_branding_design_tokens.md  design-token PATCH on global branding
│   └── paywall_route.md                 public /paywall/{app_slug} upgrade page
├── lifecycle/
│   ├── state_machine.md           — Stopped/Running/Reloading transitions
│   └── reload_behavior.md         — race handling + engine refresh semantics
├── contracts/
│   ├── decision_kinds.md          — Allow / DenyAnonymous / DenyForbidden / PaywallRedirect
│   ├── m2m_rule_flag.md           — M2M bool field + evaluation order
│   ├── require_grammar.md         — every accepted Require/Allow value
│   └── rule_shape.md              — full RuleSpec JSON schema
├── migration/
│   ├── yaml_deprecation.md        — moving proxy.rules YAML into the DB
│   ├── 00023_tier_match.md        — proxy_rules.tier_match column
│   ├── 00024_m2m.md               — proxy_rules.m2m column
│   └── 00025_branding_design_tokens.md — branding.design_tokens column
└── integration/
    ├── frontend_wiring_notes.md   — dashboard-tab blueprints
    └── sdk_surface.md             — TS + Python SDK method projection
```

## How to consume this docs set

**Dashboard authors:** Start with `integration/frontend_wiring_notes.md`. Each bullet names a dashboard surface (Proxy tab, lifecycle toggle, tier dropdown, design-tokens editor, paywall preview, YAML import) and links the backing API doc. Drop into the `api/*.md` files for the exact request/response shapes, then read the relevant `contracts/*.md` if your UI surfaces decision kinds, require predicates, or the m2m flag.

**SDK authors:** Start with `integration/sdk_surface.md`. It enumerates every endpoint with the target TS + Python method name + signature. Each row links back to the `api/*.md` file that owns the schema. The contract docs double as SDK type references — `rule_shape.md` is what you'd marshal into `ProxyRule`, `decision_kinds.md` is the enum your simulate/evaluate helpers return.

**Operators:** Start with `migration/yaml_deprecation.md`. It explains the v1.5 → v2 timeline (v1.5 warns on legacy YAML rules; v2 will reject), the exact curl / CLI payloads that translate a `proxy.rules:` YAML block into DB rows, and the bulk-import endpoint for zero-downtime migration.

**Auth:** Every route under `/api/v1/admin/*` is gated by the admin API key middleware (`AdminAPIKeyFromStore`). The paywall route and `/assets/branding/*` are intentionally public. See each `api/*.md` for per-route auth detail.
