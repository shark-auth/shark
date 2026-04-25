# proxy_lifecycle.py

**Path:** `sdk/python/shark_auth/proxy_lifecycle.py`
**Module:** `shark_auth.proxy_lifecycle`
**LOC:** 81

## Purpose
Admin client for v1.5 proxy lifecycle control — start, stop, reload, and live-status snapshot of the in-process proxy Manager.

## Public API
- `class ProxyStatus(TypedDict)`
  - `state: int`
  - `state_str: Literal["stopped", "running", "reloading", "unknown"]`
  - `listeners: int`
  - `rules_loaded: int`
  - `started_at: str` — ISO timestamp
  - `last_error: str`
- `class ProxyLifecycleClient`
  - `__init__(base_url, token, *, session=None)`
  - `.get_proxy_status() -> ProxyStatus` — `GET /api/v1/admin/proxy/lifecycle`
  - `.start_proxy() -> ProxyStatus` — `POST /api/v1/admin/proxy/start`
  - `.stop_proxy() -> ProxyStatus` — `POST /api/v1/admin/proxy/stop` (idempotent)
  - `.reload_proxy() -> ProxyStatus` — `POST /api/v1/admin/proxy/reload`

## Internal helpers
- `._post(path)` / `._get(path)` — accept either `{data: status}` or bare status payloads; route non-200 through `_raise()`
- `._auth()` — `Bearer <admin token>`

## Internal dependencies
- `_http.new_session`, `_http.request`
- `proxy_rules.SharkAPIError`, `proxy_rules._raise`

## Notes
- `stop_proxy()` is idempotent server-side — a stop on a stopped proxy returns 200.
- `reload_proxy()` is implemented as stop+start in a single critical section on the server side, so callers see one atomic transition.
- `state_str` is the only stable representation; `state` (int) may shift between server versions.
- All four methods share the same `ProxyStatus` shape so callers can chain calls and inspect transitions uniformly.
