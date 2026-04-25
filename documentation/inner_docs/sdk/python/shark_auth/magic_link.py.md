# magic_link.py

**Path:** `sdk/python/shark_auth/magic_link.py`

Magic link send client for the Python SDK. Public endpoint — no admin key required.

## Exports

- `MagicLinkClient` — class

## Constructor

```python
MagicLinkClient(base_url: str, *, session=None)
```

- `base_url` — SharkAuth server URL
- `session` — optional pre-configured `requests.Session`

## Methods

### `send_magic_link(email, redirect_uri=None) -> dict`

POSTs `{email[, redirect_uri]}` to `POST /api/v1/auth/magic-link/send`. The server applies per-email rate limiting (1 per 60 s) and always returns 200 to avoid leaking account existence. Returns the JSON body (e.g. `{"message": "..."}`) on success. Raises `SharkAPIError` on non-200.

`redirect_uri` must be on the server's `allowed_callback_urls` allowlist.

## Route

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/v1/auth/magic-link/send` | None |

## Notes

- 503 `feature_disabled` returned when magic link is not configured (email provider absent).
- Anti-enumeration: server returns 200 regardless of whether the email has an account.
