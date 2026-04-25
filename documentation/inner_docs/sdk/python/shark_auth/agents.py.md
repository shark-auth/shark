# agents.py

**Path:** `sdk/python/shark_auth/agents.py`
**Module:** `shark_auth.agents`
**LOC:** 111

## Purpose
Admin client for managing machine-to-machine agents (OAuth 2.1 clients) — register, list, revoke. Wraps the `/api/v1/agents` admin routes.

## Public API
- `class AgentsClient`
  - `_PREFIX = "/api/v1/agents"`
  - `__init__(base_url, token, *, session=None)`
  - `.register_agent(app_id, name, scopes=None, **extra) -> dict` — `POST /api/v1/agents`; body merges `{name, scopes, metadata: {app_id}}` with `**extra` (e.g. `description`, `token_lifetime`, `redirect_uris`); response includes one-time `client_secret`
  - `.list_agents(app_id: str | None = None) -> list[dict]` — `GET /api/v1/agents[?search=app_id]`; returns `body["data"]`
  - `.revoke_agent(agent_id: str) -> None` — `DELETE /api/v1/agents/{id}`; expects 204

## Constructor params
- `base_url: str` — required
- `token: str` — admin API key (`sk_live_…`)
- `session: object | None` — optional shared `requests.Session`

## Internal dependencies
- `_http.new_session`, `_http.request`
- `proxy_rules._raise` — shared SharkAPIError raiser

## Notes
- `register_agent` accepts both 200 and 201 as success.
- The one-time `client_secret` returned at registration is never re-fetched — callers must store it immediately.
- `list_agents` filters via the server's `?search=` parameter (matches name and id), not a strict `app_id` filter — narrow further client-side if you need exact app scoping.
- Plain dicts everywhere — no models.
