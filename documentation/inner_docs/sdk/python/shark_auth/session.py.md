# session.py

**Path:** `sdk/python/shark_auth/session.py`
**Module:** `shark_auth.session`
**LOC:** 52

## Purpose
A `requests.Session` subclass that auto-signs every outgoing request with a DPoP proof bound to the access token — so caller code looks like vanilla `requests` but every call satisfies RFC 9449.

## Public API
- `class AgentSession(requests.Session)`
  - `__init__(prover: DPoPProver, access_token: str, *, user_agent="shark-auth-python/0.1.0")`
  - `.prover: DPoPProver`
  - `.access_token: str`
  - `.request(method, url, *args, **kwargs)` — overridden; computes proof for `(method, url)` with `ath = sha256(access_token)`, then injects:
    - `Authorization: DPoP <access_token>`
    - `DPoP: <proof>`

## Constructor params
- `prover: DPoPProver` — required; the keypair that bound the access token
- `access_token: str` — required; Shark-issued agent token
- `user_agent: str` — keyword-only; sets `User-Agent` and `Accept: application/json` defaults

## Internal dependencies
- `requests.Session` (parent class)
- `dpop.DPoPProver`

## Notes
- A fresh proof is generated for every request (correct: `htu`/`htm`/`jti` must be unique per RFC 9449 §4.2).
- The override is on `request()`, so every shorthand (`.get`, `.post`, `.put`, `.patch`, `.delete`) flows through it.
- No nonce handling — if the server returns `DPoP-Nonce`, callers must re-issue at a higher layer (planned).
- The class is exported as `AgentSession` from the package root.
- Drop-in replacement for `requests.Session()` — works with adapters, mounts, cookies, etc.
