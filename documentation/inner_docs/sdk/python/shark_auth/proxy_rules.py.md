# proxy_rules.py

**Path:** `sdk/python/shark_auth/proxy_rules.py`
**Module:** `shark_auth.proxy_rules`
**LOC:** 201

## Purpose
DB-backed proxy-rules CRUD against the v1.5 admin API (`/api/v1/admin/proxy/rules/db`) plus YAML bulk import. Also defines `SharkAPIError` and the `_raise()` helper that the rest of the admin sub-clients reuse.

## Public API
- `class SharkAPIError(SharkAuthError)` — `.code`, `.status` attributes
- TypedDicts:
  - `ProxyRule` — full row: `id, app_id, name, pattern, methods, require, allow, scopes, enabled, priority, tier_match, m2m, created_at, updated_at`
  - `CreateProxyRuleInput` — `total=False`; same fields minus timestamps and id
  - `UpdateProxyRuleInput = CreateProxyRuleInput` (alias)
  - `ImportResult` — `imported: int`, `errors: list[dict]`
- `_raise(resp)` — module-level helper that unwraps `{error: {code, message}}` envelopes and raises `SharkAPIError`
- `class ProxyRulesClient` (`_PREFIX = "/api/v1/admin/proxy/rules"`)
  - `__init__(base_url, token, *, session=None)`
  - `.list_rules(app_id=None) -> list[ProxyRule]` — `GET …/db`
  - `.create_rule(spec: CreateProxyRuleInput, id=None) -> ProxyRule` — `POST …/db`
  - `.get_rule(rule_id) -> ProxyRule` — `GET …/db/{id}`
  - `.update_rule(rule_id, **patch) -> ProxyRule` — `PATCH …/db/{id}`
  - `.delete_rule(rule_id) -> None` — `DELETE …/db/{id}`; expects 204
  - `.import_rules_yaml(yaml_text: str) -> ImportResult` — `POST /api/v1/admin/proxy/rules/import`

## Internal dependencies
- `_http.new_session`, `_http.request`
- `errors.SharkAuthError` (parent of `SharkAPIError`)

## Notes
- `_raise()` is reused by `users.py`, `agents.py`, `branding.py`, `paywall.py`, and `proxy_lifecycle.py` — single source of truth for admin error envelopes.
- TypedDicts give static type safety without pulling in Pydantic.
- `import_rules_yaml` partial-success: `imported > 0` does not preclude per-row `errors` — always inspect both.
- Trailing slash on `base_url` is auto-stripped.
