# Integration — Frontend wiring notes

Blueprints for the dashboard surfaces that consume the v1.5 proxy admin API. Each bullet names a surface, cites the exact endpoint, and links the backing docs.

## Proxy tab — rules list + editor

**Endpoints:**
- `GET    /api/v1/admin/proxy/rules/db`      — list (see `api/admin_proxy_rules_db.md`)
- `POST   /api/v1/admin/proxy/rules/db`      — create
- `GET    /api/v1/admin/proxy/rules/db/{id}` — fetch one
- `PATCH  /api/v1/admin/proxy/rules/db/{id}` — update
- `DELETE /api/v1/admin/proxy/rules/db/{id}` — delete

**UI:**
- Table view, priority DESC. Columns: name, pattern, methods, require/allow badge, tier_match chip, m2m chip, enabled toggle, priority, updated_at.
- Inline row actions: edit, delete, toggle enabled. Delete confirms with a modal.
- Create/edit modal renders a form over the full `ProxyRule` shape (see `contracts/rule_shape.md`). Pre-validate `require` against the grammar in `contracts/require_grammar.md`.
- Surface `engine_refresh_error` from POST/PATCH responses as an inline warning on the save toast — the DB write succeeded but the live engine is stale.

## Proxy tab — lifecycle toggle + status indicator

**Endpoints:**
- `GET  /api/v1/admin/proxy/lifecycle` — status (see `api/admin_proxy_lifecycle.md`)
- `POST /api/v1/admin/proxy/start`     — start
- `POST /api/v1/admin/proxy/stop`      — stop
- `POST /api/v1/admin/proxy/reload`    — reload

**UI:**
- Header chip: "running / stopped / reloading" with color + pulse animation during `reloading`.
- Split button: "Start / Stop / Reload". Disabled-state derived from `state_str` (can't Start when already Running, etc.).
- Last-error banner when `last_error != ""`. Clickable to reveal full message.
- Optional SSE subscription to `/api/v1/admin/proxy/status/stream` for real-time state updates without polling. Fall back to 5s poll when SSE unavailable.
- 404 response on any of these routes means the proxy Manager wasn't wired (proxy disabled at boot). Hide the entire lifecycle panel in that case.

## User detail — tier dropdown

**Endpoint:** `PATCH /api/v1/admin/users/{id}/tier` (see `api/admin_users_tier.md`)

**UI:**
- Segmented control on the user detail page: Free / Pro.
- Optimistic update is fine — error surface is narrow (unknown tier, 404).
- Helper text: "Access tokens carry the tier at issue time; force a logout for immediate cutover." Link to the existing sessions-revoke action.
- Audit log entry is auto-created; no dashboard-side audit work needed.

## Branding page — design tokens editor

**Endpoint:** `PATCH /api/v1/admin/branding/design-tokens` (see `api/admin_branding_design_tokens.md`)

**UI:**
- Split pane: left = JSON editor (CodeMirror or Monaco in JSON mode) with schema validation; right = live preview fragment rendered with the current tokens as CSS custom properties.
- "Reset to defaults" button clears the column (send `{}`).
- "Copy as CSS vars" secondary action for operators who want to hand-copy tokens into their own stylesheets.
- Read the current value from `GET /api/v1/admin/branding` (existing endpoint) — response includes `design_tokens` after migration 00025.

## Branding page — paywall preview

**Endpoint:** `GET /paywall/{app_slug}` (see `api/paywall_route.md`)

**UI:**
- "Preview paywall" action on the Branding / Applications page.
- Opens an iframe (or new tab) pointing at `/paywall/<current_app_slug>?tier=pro&return=/`.
- Pair with a tier dropdown (`tier=free | pro`) so designers can preview each tier's copy.
- No SDK call needed — this is a pure URL navigation.

## Import YAML rules

**Endpoint:** `POST /api/v1/admin/proxy/rules/import` (see `api/admin_proxy_rules_import.md`)

**UI:**
- Drag-and-drop zone + textarea fallback on the Proxy tab.
- Submit as JSON `{ "yaml": "<file contents>" }`.
- Render `errors[]` as a two-column table (index + name + message). Surface `imported` as a success toast.
- Disable the "delete original YAML" button until `errors.length === 0` — partial success is the contract.
- Link to `migration/yaml_deprecation.md` below the form so first-time migrators can self-serve.

## Global: auth + error envelope

Every `/admin/*` route is gated by admin API key (`Authorization: Bearer sk_live_...`). The dashboard should inject it from the session store on every request.

Error envelope is consistent across routes:

```json
{ "error": { "code": "invalid_proxy_rule", "message": "..." } }
```

Global handler: map `error.code` to a user-facing toast + optional inline field error. Field-error wiring lives in the request-level handlers (e.g. `invalid_proxy_rule` → show the message under the form).
