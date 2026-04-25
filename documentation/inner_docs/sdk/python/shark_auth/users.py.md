# users.py

**Path:** `sdk/python/shark_auth/users.py`
**Module:** `shark_auth.users`
**LOC:** 93

## Purpose
Admin client wrapping the v1 `/api/v1/users` endpoints plus the v1.5 tier-management route — list users, fetch by id, set billing tier.

## Public API
- `class UsersClient`
  - `__init__(base_url, token, *, session=None)`
  - `.list_users(email=None) -> list[dict]` — `GET /api/v1/users[?email=…]`; tolerates `{data:[…]}` or bare list
  - `.get_user(user_id) -> dict` — `GET /api/v1/users/{id}`
  - `.create_user(email, *, password=None, name=None, email_verified=False) -> dict` — `POST /api/v1/admin/users`; admin-only create with optional pre-verified email
  - `.update_user(user_id, *, email=None, name=None, email_verified=None, metadata=None) -> dict` — `PATCH /api/v1/users/{id}`; partial update, only supplied fields changed
  - `.delete_user(user_id) -> None` — `DELETE /api/v1/users/{id}`; 204 on success
  - `.set_user_tier(user_id, tier: Literal["free", "pro"]) -> dict` — `PATCH /api/v1/admin/users/{id}/tier`; returns `{user, tier}`

## Constructor params
- `base_url: str` — required
- `token: str` — admin API key (`sk_live_…`)
- `session: object | None` — optional shared `requests.Session`

## Internal dependencies
- `_http.new_session`, `_http.request`
- `proxy_rules._raise` — shared error envelope unwrapper that raises `SharkAPIError`

## Notes
- `tier` is statically typed `Literal["free", "pro"]`; the server enforces the same set with HTTP 400 on invalid values.
- Returns plain `dict` (no Pydantic models) — server response shape passes through verbatim.
- All non-2xx responses go through `_raise()` and surface as `SharkAPIError(code, status, message)`.
- This client is one of six accessed via the unified `Client` namespace (`c.users.…`).
