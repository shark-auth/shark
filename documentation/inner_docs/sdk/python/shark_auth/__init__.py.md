# __init__.py

**Path:** `sdk/python/shark_auth/__init__.py`
**Module:** `shark_auth`
**LOC:** 75

## Purpose
Public package surface — re-exports every documented class, function, and exception so consumers can `from shark_auth import …` without reaching into private submodules.

## Public API (re-exports)
v1.5 admin client + sub-clients:
- `Client` — unified admin client (from `.client`)
- `ProxyRulesClient` (from `.proxy_rules`)
- `ProxyLifecycleClient` (from `.proxy_lifecycle`)
- `BrandingClient` (from `.branding`)
- `PaywallClient` (from `.paywall`)
- `UsersClient` (from `.users`)
- `AgentsClient` (from `.agents`)

v1.5 types:
- `ProxyRule`, `CreateProxyRuleInput`, `UpdateProxyRuleInput`, `ImportResult` (TypedDicts)
- `ProxyStatus` (TypedDict)
- `SharkAPIError`

Pre-existing primitives:
- `DPoPProver` (RFC 9449)
- `AgentSession` (requests.Session subclass with auto DPoP)
- `DeviceFlow`, `DeviceInit`, `TokenResponse` (RFC 8628)
- `VaultClient`, `VaultToken`
- `AgentTokenClaims`, `decode_agent_token`, `exchange_token`, `clear_jwks_cache`

Errors:
- `SharkAuthError` (base)
- `DPoPError`, `DeviceFlowError`, `VaultError`, `TokenError`

Constants:
- `__version__ = "0.1.0"`

## Notes
- `__all__` mirrors all of the above explicitly (PEP 8 / star-import safe).
- Module docstring lists the five most prominent public symbols at the top — useful for IDE intellisense and `help(shark_auth)`.
- No side effects on import — all sub-modules are pure definitions.
