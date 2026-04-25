# oauth.py

**Path:** `sdk/python/shark_auth/oauth.py`

RFC 7009 token revocation and RFC 7662 token introspection client for the Python SDK.

## Exports

- `OAuthClient` — class

## Constructor

```python
OAuthClient(base_url: str, token: str = "", *, session=None)
```

- `base_url` — SharkAuth server URL
- `token` — admin API key (`sk_live_...`); required for introspection, optional for revocation
- `session` — optional pre-configured `requests.Session`

## Methods

### `revoke_token(token, token_type_hint=None) -> None`

RFC 7009. Posts form-encoded `{token[, token_type_hint]}` to `POST /oauth/revoke`. Server always returns 200 regardless of whether the token existed. Raises `SharkAPIError` on non-200.

### `introspect_token(token) -> dict`

RFC 7662. Posts form-encoded `{token}` to `POST /oauth/introspect`. Returns dict with `active: bool` and claims when active. Returns `{"active": False}` for invalid/expired tokens.

## Routes

| Method | Path | Auth |
|--------|------|------|
| POST | `/oauth/revoke` | None required (Bearer forwarded if set) |
| POST | `/oauth/introspect` | Bearer admin key |

## Notes

- Revocation is always safe to call — server never reveals whether the token existed.
- 503 `feature_disabled` is returned when the nil-config path is hit (OAuth AS not configured).
