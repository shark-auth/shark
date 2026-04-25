# client.py

**Path:** `sdk/python/shark_auth/client.py`
**Module:** `shark_auth.client`
**LOC:** 67

## Purpose
Unified `Client` namespace that constructs all v1.5 admin sub-clients with a shared `requests.Session` for connection pooling — mirrors the TypeScript SDK Lane F surface.

## Public API
- `class Client`
  - `__init__(base_url, token, *, session=None)` — construct
  - `.proxy_rules` — `ProxyRulesClient` (DB-backed CRUD)
  - `.proxy_lifecycle` — `ProxyLifecycleClient` (start/stop/reload/status)
  - `.branding` — `BrandingClient` (design tokens)
  - `.paywall` — `PaywallClient` (URL builder + HTML fetch)
  - `.users` — `UsersClient` (list/get/tier)
  - `.agents` — `AgentsClient` (register/list/revoke)
  - `__repr__()` — `Client(base_url=…)`

## Constructor params
- `base_url: str` — required; trailing slash auto-stripped
- `token: str` — required admin API key (`sk_live_…`)
- `session: object | None` — optional shared `requests.Session`; one is created via `_http.new_session()` when omitted

## Internal dependencies
- `_http.new_session` — shared connection pool
- `agents.AgentsClient`, `branding.BrandingClient`, `paywall.PaywallClient`,
  `proxy_lifecycle.ProxyLifecycleClient`, `proxy_rules.ProxyRulesClient`, `users.UsersClient`

## Notes
- Synchronous; no async variant yet.
- This `Client` is the v1.5 admin entrypoint only — DPoP / device flow / vault / token-exchange remain top-level helpers (`DPoPProver`, `DeviceFlow`, `VaultClient`, `exchange_token`).
- All sub-clients share the same session, so HTTP keep-alive works across namespaces.
- Docstring includes a runnable example: `c.proxy_rules.list_rules()` and `c.proxy_lifecycle.get_proxy_status()`.
