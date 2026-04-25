# tokens.py

**Path:** `sdk/python/shark_auth/tokens.py`
**Module:** `shark_auth.tokens`
**LOC:** 260

## Purpose
Two responsibilities: (1) verify Shark-issued agent access tokens via JWKS with kid-miss refresh, and (2) perform RFC 8693 token exchange so an agent can act on a user's behalf.

## Public API
- Constants: `TOKEN_EXCHANGE_GRANT`, `ACCESS_TOKEN_TYPE`
- `@dataclass class AgentTokenClaims`
  - `sub, iss, aud, exp, iat, scope?, agent_id?, act?, cnf?, authorization_details?, raw`
  - `.jkt` property — pulls `cnf.jkt` (DPoP confirmation thumbprint) when present
- `decode_agent_token(token, jwks_url, *, expected_issuer, expected_audience, leeway=0) -> AgentTokenClaims` — verifies sig + `exp`/`nbf` + `iss`/`aud`; requires `exp, iat, iss, aud, sub` claims
- `exchange_token(auth_url, client_id, subject_token, *, client_secret=None, subject_token_type=ACCESS_TOKEN_TYPE, actor_token=None, actor_token_type=ACCESS_TOKEN_TYPE, scope=None, requested_token_type=ACCESS_TOKEN_TYPE, dpop_prover=None, token_path="/oauth/token", session=None) -> TokenResponse` — RFC 8693
- `clear_jwks_cache() -> None` — testing hook

## JWKS cache
- Module-level dict keyed by `jwks_url`, guarded by `threading.Lock`.
- On `kid` miss, performs a single forced refresh before erroring.

## Errors raised
- `TokenError` — malformed header, missing `kid`, `alg=none`, no JWK found, expired, invalid issuer/audience/signature, JWKS fetch failure.

## Internal dependencies
- `PyJWT` (`PyJWKClient`, `PyJWKSet`, `jwt.decode`)
- `_http.new_session`, `_http.request`
- `device_flow.TokenResponse`
- `dpop.DPoPProver` (optional, for DPoP-bound exchange)
- `errors.TokenError`

## Notes
- Preserves all RFC 8693 (`act`), RFC 9449 (`cnf`), and RFC 9396 (`authorization_details`) claims via `AgentTokenClaims`.
- `decode_agent_token` rejects `alg=none` explicitly.
- The exchange leg supports both confidential (`client_secret`) and public (DPoP-only) clients.
