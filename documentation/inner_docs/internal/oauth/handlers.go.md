# handlers.go

**Path:** `internal/oauth/handlers.go`
**Package:** `oauth`
**LOC:** 387
**Tests:** `handlers_test.go`

## Purpose
HTTP handlers for the standard OAuth 2.1 endpoints (`/token`, `/authorize`, `/authorize/decision`) including grant-type dispatch and DPoP interception.

## RFCs implemented
- RFC 6749 token + authorize endpoints
- RFC 7636 PKCE (passed through fosite)
- RFC 8628 device-grant interception (line 49)
- RFC 8693 token-exchange interception (line 56)
- RFC 8707 resource-indicator capture (line 87)
- RFC 9449 DPoP proof validation at /token (line 64)

## Key types / functions
- `dpopTokenEndpointURL` (func, line 26) — canonical HTU builder (strips query/fragment).
- `HandleToken` (func, line 47) — grant_type dispatcher. Routes device_code → `HandleDeviceTokenRequest`, token-exchange → `HandleTokenExchange`, otherwise validates DPoP then hands to fosite. Enriches JWT claims (`client_id`, `cnf.jkt`, `jti`) before signing.
- `HandleAuthorize` (func, line 181) — parses authorize request, gates on session middleware, redirects to hosted login or renders consent page.
- `HandleAuthorizeDecision` (func, later) — POST handler that records consent and finalises the authorize response.

## Imports of note
- `github.com/ory/fosite`, `github.com/ory/fosite/token/jwt`
- `internal/api/middleware` (session lookup)
- `internal/storage`, `github.com/google/uuid`

## Wired by
- `internal/server/server.go` — mounts `POST /oauth/token`, `GET /oauth/authorize`, `POST /oauth/authorize/decision`.

## Used by
- All OAuth clients (web SDK, agents, MCP).

## Notes
- DPoP failure short-circuits to `invalid_dpop_proof` JSON before fosite touches the request.
- Audience fallback (line 113) grants `client_id` as aud when no `resource`/`audience` requested — Auth0/Okta convention.
- `jc.Subject` set to client_id only on the JWT claims, never the DefaultSession (avoids `oauth_tokens.user_id` FK violation).
