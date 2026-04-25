# _http.py

**Path:** `sdk/python/shark_auth/_http.py`
**Module:** `shark_auth._http`
**LOC:** 28

## Purpose
Tiny shared HTTP layer on top of `requests` — provides a default-User-Agent session and a thin `request()` wrapper that enforces a default 10-second timeout.

## Public API
Module-private (leading underscore), but used internally by every sub-client:
- `DEFAULT_TIMEOUT = 10.0` — seconds
- `new_session(user_agent="shark-auth-python/0.1.0") -> requests.Session` — pre-configured session with `User-Agent` and `Accept: application/json` headers
- `request(session, method, url, *, timeout=None, **kwargs) -> requests.Response` — wraps `session.request()` and applies `DEFAULT_TIMEOUT` when `timeout` is not supplied

## Internal dependencies
- `requests` (third-party)

## Notes
- Timeout applies to the entire request (connect + read).
- `**kwargs` is forwarded verbatim to `session.request()` — sub-clients pass `headers`, `json`, `data`, `params`, etc.
- No retry / backoff logic — that's a caller concern.
- No async variant; the SDK is synchronous-only today.
- The `User-Agent` string is bumped manually on version changes (no automatic sync from `__version__`).
- The leading underscore signals "package-private" but nothing prevents external import.
