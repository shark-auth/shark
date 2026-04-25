# agents.ts

**Path:** `sdk/typescript/src/agents.ts`
**Type:** Admin namespace — agent OAuth client management
**LOC:** 186

## Purpose
Register, list, fetch, and revoke agent OAuth clients (the SharkAuth equivalent of an Auth0 application).

## Public API
- `class AgentsClient`
  - `constructor(opts: AgentsClientOptions)`
  - `registerAgent(input: RegisterAgentInput): Promise<AgentCreateResult>` — POST `/api/v1/agents`, expects 201
  - `listAgents(opts?: ListAgentsOptions): Promise<AgentListResult>` — GET with `active`, `search`, `limit`, `offset`
  - `getAgent(id): Promise<Agent>` — GET by ID or clientID
  - `revokeAgent(id): Promise<void>` — DELETE, expects 204 — sets `active=false` and revokes all outstanding tokens for `client_id`

## Agent shape
`id, name, description?, client_id, client_type, auth_method, redirect_uris[], grant_types[], response_types[], scopes[], token_lifetime, metadata, logo_uri?, homepage_uri?, active, created_at, updated_at`

## AgentCreateResult
Extends `Agent` with `client_secret: string` — plaintext, shown **exactly once**, server stores only the hash.

## RegisterAgentInput
Required: `name`. Defaults: `client_type="confidential"`, `auth_method="client_secret_basic"`, `token_lifetime=3600`. All OAuth metadata fields optional.

## ListAgentsOptions
`{ active?, search?, limit?, offset? }` — `search` matches name.

## Constructor options
- `baseUrl: string`
- `adminKey: string` — Bearer token

## Error mapping
- Non-success → `SharkAPIError` parsed from `{error:{code,message}}`.

## Internal dependencies
- `http.ts`, `errors.ts`

## Notes
- Revocation is a soft delete — record stays for audit, but `active=false`.
- Capture `client_secret` immediately on `registerAgent`; not retrievable later.
