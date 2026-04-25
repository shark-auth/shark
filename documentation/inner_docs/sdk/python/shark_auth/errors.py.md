# errors.py

**Path:** `sdk/python/shark_auth/errors.py`
**Module:** `shark_auth.errors`
**LOC:** 27

## Purpose
Exception hierarchy for the SDK — one base class + four typed subclasses so callers can `except` at the granularity they need.

## Public API
- `class SharkAuthError(Exception)` — root of all SDK exceptions
- `class DPoPError(SharkAuthError)` — DPoP proof construction / signing failure
- `class DeviceFlowError(SharkAuthError)` — RFC 8628 device flow failure (denied, expired, timeout, bad client)
- `class VaultError(SharkAuthError)` — token-vault interaction failure
  - `__init__(message, status_code: int | None = None)` — preserves HTTP status
  - `.status_code` attribute (e.g. 401, 403, 404)
- `class TokenError(SharkAuthError)` — JWT decode / verify failure (signature, expiry, issuer, audience, missing JWK)

## Notes
- `SharkAPIError` (raised by admin sub-clients) lives in `proxy_rules.py`, not here — but it does subclass `SharkAuthError`, so `except SharkAuthError` catches everything.
- All classes are pure Python `Exception` subclasses — no metaclasses, no Pydantic.
- `VaultError` is the only one with a payload field; the others rely on `str(exc)` for context.
- Error messages typically include the upstream HTTP status and a truncated response body where relevant.
- These exceptions are imported and re-exported by `shark_auth/__init__.py`.
