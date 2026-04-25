# vault.py

**Path:** `sdk/python/shark_auth/vault.py`
**Module:** `shark_auth.vault`
**LOC:** 81

## Purpose
Shark Token Vault client — fetches fresh, server-refreshed third-party OAuth access tokens (Google, GitHub, etc.) for a stored connection so agents never handle long-lived refresh tokens directly.

## Public API
- `@dataclass class VaultToken`
  - `access_token: str`
  - `expires_at: int | None` — unix seconds
  - `provider: str | None`
  - `scopes: list[str]`
- `class VaultClient`
  - `__init__(auth_url, access_token, *, session=None, connections_path="/admin/vault/connections")`
  - `.get_fresh_token(connection_id: str) -> VaultToken` — `GET {connections_path}/{id}/token`

## HTTP behavior
- Auth: `Authorization: Bearer <access_token>` (the agent's Shark-issued token).
- Status mapping:
  - 200 → `VaultToken`
  - 404 → `VaultError("connection not found", 404)`
  - 401 → `VaultError("agent not authorized", 401)`
  - 403 → `VaultError("missing scope for vault access", 403)`
  - other → `VaultError(...)` with truncated body
- Empty `connection_id` raises `VaultError` before the request.
- `scopes` accepts list, space-separated string, or missing — normalized to `list[str]`.

## Internal dependencies
- `_http.new_session`, `_http.request`
- `errors.VaultError`

## Notes
- The vault endpoint refreshes the upstream token server-side; callers always receive a token guaranteed valid at fetch time (refresh races are upstream's problem).
- No cache — every call hits the server. Cache at the caller layer if you need it.
- Path is configurable to support alternate mounts.
