# exchange.ts

**Path:** `sdk/typescript/src/exchange.ts`
**Type:** RFC 8693 token exchange (delegation)
**LOC:** 155

## Purpose
Performs an OAuth 2.0 RFC 8693 Token Exchange so an agent can act on behalf of a user (delegation chain). The agent supplies the user's `subject_token`; the server returns a new token bound to the agent as the actor.

## Public API
- `function exchangeToken(opts: TokenExchangeOptions): Promise<TokenResponse>`
- `interface TokenExchangeOptions`

## Constants used
- `grant_type` = `urn:ietf:params:oauth:grant-type:token-exchange`
- Default `subject_token_type`/`actor_token_type` = `urn:ietf:params:oauth:token-type:access_token`
- Other accepted token types: `refresh_token`, `jwt`

## Options
- `authUrl: string`
- `clientId: string` — acting agent
- `clientSecret?: string` — for confidential clients
- `subjectToken: string` — token representing the identity being acted upon (required)
- `subjectTokenType?: string` — defaults to access-token URN
- `actorToken?: string` — actor's own identity if delegation chain is explicit
- `actorTokenType?: string`
- `scope?: string` — space-delimited
- `requestedTokenType?: string`
- `dpopProver?: DPoPProver` — adds `DPoP` header; result token will be DPoP-bound
- `tokenPath?: string` — default `/oauth/token`

## Returns `TokenResponse` from deviceFlow.ts
- `accessToken`, `tokenType` (default `Bearer`), `expiresIn?`, `refreshToken?`, `scope?`

## Error handling
- Non-200 + JSON `error` field → `TokenError("token exchange failed: <error> (HTTP <status>): <desc>")`
- Non-JSON body → `TokenError("token endpoint returned non-JSON…")`

## Internal dependencies
- `http.ts`, `dpop.ts`, `errors.ts`
- Reuses `TokenResponse` from `deviceFlow.ts`

## Notes
- Standalone function — not a class. Stateless.
- Use case: agent worker exchanges a user JWT for `act` claim chains across services.
