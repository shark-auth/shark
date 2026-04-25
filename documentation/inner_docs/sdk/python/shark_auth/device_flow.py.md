# device_flow.py

**Path:** `sdk/python/shark_auth/device_flow.py`
**Module:** `shark_auth.device_flow`
**LOC:** 180

## Purpose
RFC 8628 OAuth 2.0 Device Authorization Grant client — initiates a device authorization request, then polls the token endpoint until the user approves, with proper handling of `authorization_pending`, `slow_down`, `access_denied`, `expired_token`.

## Public API
- `DEVICE_GRANT = "urn:ietf:params:oauth:grant-type:device_code"`
- `@dataclass class DeviceInit`
  - `device_code, user_code, verification_uri, verification_uri_complete, expires_in, interval`
- `@dataclass class TokenResponse`
  - `access_token, token_type, expires_in?, refresh_token?, scope?`
- `class DeviceFlow`
  - `__init__(auth_url, client_id, scope=None, *, dpop_prover=None, device_authorization_path="/oauth/device_authorization", token_path="/oauth/token", session=None)`
  - `.begin() -> DeviceInit` — POSTs to device-auth endpoint
  - `.wait_for_approval(timeout_s=300.0, *, clock=None, sleeper=None) -> TokenResponse` — polls token endpoint

## Polling behavior
- Default poll interval taken from server `interval` (fallback 5s).
- `slow_down` increases interval by 5s per RFC 8628 §3.5.
- Test hooks: `clock` (for monotonic) and `sleeper` enable deterministic tests.
- DPoP: when `dpop_prover` is supplied, every poll attaches a `DPoP:` header signed for `POST <token_url>`.

## Errors raised
- `DeviceFlowError` — non-200 device-auth response, missing required fields, `access_denied`, `expired_token`, `invalid_client`, polling timeout, non-JSON token response.

## Internal dependencies
- `_http.new_session`, `_http.request`
- `dpop.DPoPProver` (optional)
- `errors.DeviceFlowError`

## Notes
- `begin()` must be called before `wait_for_approval()`; otherwise `DeviceFlowError`.
- `DeviceInit` is cached on the instance after `begin()`.
- Endpoint paths are configurable to support non-default routes.
