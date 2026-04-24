# Admin API — DB-backed proxy rules CRUD

## Purpose

DB-backed proxy rules are the v1.5 source of truth for the reverse-proxy authorization engine. The CRUD surface lets admins list, create, read, update, and delete rules without restarting the server or editing YAML; every mutation refreshes the in-process rule engine via `Engine.SetRules` so the change takes effect on the next request.

## Routes

| Method | Path | Handler symbol |
|---|---|---|
| GET    | `/api/v1/admin/proxy/rules/db`           | `Server.handleListProxyRules`  |
| POST   | `/api/v1/admin/proxy/rules/db`           | `Server.handleCreateProxyRule` |
| GET    | `/api/v1/admin/proxy/rules/db/{id}`      | `Server.handleGetProxyRule`    |
| PATCH  | `/api/v1/admin/proxy/rules/db/{id}`      | `Server.handleUpdateProxyRule` |
| DELETE | `/api/v1/admin/proxy/rules/db/{id}`      | `Server.handleDeleteProxyRule` |

## Auth required

Admin API key (via `Authorization: Bearer sk_live_<...>`). Inherits the `AdminAPIKeyFromStore` middleware from the parent `/admin` group.

## Request shapes

### List (GET)

Query parameter:

- `app_id` — optional; when set, returns only rules scoped to that application. Omit for "global rules only" (behaviour-wise: `ListProxyRules` returns every row; the engine refresh path later filters by AppID).

### Create (POST)

```json
{
  "app_id": "app_abc",          // string, optional — scopes rule to one app
  "name": "block-unauth-write", // string, required — human-readable label
  "pattern": "/api/writes/*",    // string, required — chi-style path pattern
  "methods": ["POST", "PUT"],   // string[], optional — empty = any method
  "require": "authenticated",   // string, optional — see require_grammar.md
  "allow": "",                  // string, optional — only "anonymous" accepted
  "scopes": ["webhooks:write"], // string[], optional — AND'd with require
  "enabled": true,              // bool, optional — defaults to true
  "priority": 100,              // int, optional — DESC-sorted; higher wins
  "tier_match": "pro",          // string, optional — see contracts/require_grammar.md
  "m2m": false                  // bool, optional — see contracts/m2m_rule_flag.md
}
```

Exactly one of `require` or `allow` must be set. See `contracts/rule_shape.md` for full field semantics and `contracts/require_grammar.md` for every accepted `require` string.

### Update (PATCH)

Every field is optional; only supplied fields are mutated. Typed as pointers in Go so `null` vs missing is distinguishable. The server re-validates the full row after applying the patch — a partial update that would land in an invalid state (e.g. both `require` and `allow`) rejects with 400 and leaves the row unchanged.

```json
{ "enabled": false, "priority": 50, "m2m": true }
```

### Delete (DELETE)

No body. `{id}` path param identifies the row.

## Response shapes

### Success

List:

```json
{
  "data": [ { ...proxyRule... }, ... ],
  "total": 3
}
```

Create (201):

```json
{
  "data": { ...proxyRule... },
  "engine_refresh_error": "optional string when live-engine refresh failed"
}
```

Get (200), Update (200): `{ "data": { ...proxyRule... } }`. Update may include `engine_refresh_error`.

Delete (204): empty body.

`proxyRule` object:

```json
{
  "id": "rule_abc",
  "app_id": "app_abc",
  "name": "block-unauth-write",
  "pattern": "/api/writes/*",
  "methods": ["POST", "PUT"],
  "require": "authenticated",
  "allow": "",
  "scopes": ["webhooks:write"],
  "enabled": true,
  "priority": 100,
  "tier_match": "pro",
  "m2m": false,
  "created_at": "2026-04-24T10:09:45Z",
  "updated_at": "2026-04-24T10:09:45Z"
}
```

### Error

```json
{ "error": { "code": "invalid_proxy_rule", "message": "pattern must start with '/'" } }
```

Error codes: `invalid_request` (bad JSON), `invalid_proxy_rule` (validation failure), `not_found` (404 on get/update/delete by id).

## Status codes

- `200 OK` — list, get, update, delete (success path of DELETE returns 204)
- `201 Created` — successful POST
- `204 No Content` — successful DELETE
- `400 Bad Request` — invalid JSON, missing required fields, conflicting require+allow, unknown require string
- `401 Unauthorized` — missing/invalid admin key
- `404 Not Found` — GET/PATCH/DELETE by unknown id
- `500 Internal Server Error` — storage failure, JSON encode failure

## Side effects

- DB writes: `proxy_rules` table (CREATE, UPDATE, DELETE). Read-only for LIST/GET.
- Engine refresh: `refreshProxyEngineFromDB` is invoked after every mutation. It loads every enabled global (`app_id==""`) rule and calls `Engine.SetRules`. Failure is surfaced as `engine_refresh_error` in the response but does not roll back the DB write — the row persists and the next successful mutation (or a `POST /api/v1/admin/proxy/reload`) re-publishes the full set.
- Audit log: `proxy.rule.created`, `proxy.rule.updated`, `proxy.rule.deleted` entries are written via `AuditLogger.Log` with `ActorType=admin`, `TargetType=proxy_rule`, `TargetID=rule.ID`.

## Frontend hint

A Proxy tab in the dashboard should render two views: a filterable list (priority DESC) backed by GET `/rules/db`, and a create/edit modal that POSTs / PATCHes using the same JSON shape. Surface the `engine_refresh_error` field inline on the save-toast — when it's non-empty the DB write succeeded but the live engine is still on the old rule set, which is the one bit of state the operator needs to know. The `tier_match` and `m2m` columns should render as chips so the single-row scan is fast; the TypeScript SDK method (`listProxyRules()`) returns the full shape so no additional round-trips are needed.
