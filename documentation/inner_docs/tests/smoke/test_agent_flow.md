# test_agent_flow_*.py — Agent flow smoke tests

**Path:** `tests/smoke/test_agent_flow_*.py`

Six smoke test files covering end-to-end OAuth 2.1 agent authorization flows. Each file is a standalone pytest module that spins up a live SharkAuth server (or targets `SHARK_URL`) and validates a complete authorization ceremony.

## Files

| File | Flow tested |
|------|-------------|
| `test_agent_flow_pkce.py` | Authorization code + PKCE (RFC 7636) — standard browser-based agent auth |
| `test_agent_flow_dcr.py` | Dynamic Client Registration (RFC 7591) — agent self-registers, then authenticates |
| `test_agent_flow_device.py` | Device authorization flow (RFC 8628) — headless/IoT agent auth |
| `test_agent_flow_dpop.py` | DPoP-bound tokens (RFC 9449) — proof-of-possession token binding |
| `test_agent_flow_refresh_rotation.py` | Refresh token rotation — rotate, verify old token rejected, new token valid |
| `test_agent_flow_token_exchange.py` | Token exchange (RFC 8693) — agent-to-agent delegation chain |

## Pattern

Each file:
1. Registers a client (or uses DCR) via the SharkAuth admin API
2. Performs the full authorization ceremony for the flow
3. Asserts token response shape and claims
4. For rotation/revocation tests: verifies the old credential is rejected

## Environment

| Var | Purpose |
|-----|---------|
| `SHARK_URL` | Base URL (default: `http://localhost:8080`) |
| `SHARK_ADMIN_KEY` | Admin API key (`sk_live_...`) |

## Notes

- These are integration/smoke tests — they require a running server, not mocked.
- Run in CI via `./smoke_test.sh` (sections 55-60 of the test suite, added at v0.9.0).
- Token exchange test uses the HTTP Basic auth pattern for the token endpoint.
