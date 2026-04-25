# agent_handlers.go

**Path:** `internal/api/agent_handlers.go`
**Package:** `api`
**LOC:** 539
**Tests:** likely covered via integration suites (no `agent_handlers_test.go` colocated)

## Purpose
CRUD for "agent" identities — OAuth confidential clients (machine actors / AI agents) registered in the SharkAuth tenant. Issues a one-time `client_secret` on create + rotate; secrets are SHA-256 hashed before persist.

## Handlers exposed
- `handleCreateAgent` (line 63) — POST `/api/v1/agents`. Issues `client_id = "shark_agent_<nanoid>"` and one-time hex `client_secret`; validates `name`; defaults `client_type=confidential`, `auth_method=client_secret_basic`, `token_lifetime=3600`. Audits `agent.created` + emits webhook.
- `handleListAgents` (line 178) — GET. Supports `?limit`, `?offset`, `?search`, `?active`.
- `handleGetAgent` (line 222) — GET `/{id}` (also matches by client_id via `getAgentByIDOrClientID`).
- `handleUpdateAgent` (line 237) — PATCH.
- `handleDeleteAgent` (line 319) — DELETE.
- `handleListAgentTokens` (line 348) — GET `/{id}/tokens`.
- `handleRevokeAgentTokens` (line 381) — POST `/{id}/tokens/revoke-all`.
- `handleAgentRotateSecret` (line 410) — POST `/{id}/rotate-secret`. Returns the fresh plaintext once.
- `handleAgentAuditLogs` (line 443) — GET `/{id}/audit`.

## Key types
- `agentCreateResponse` (line 21) — embeds `storage.Agent` + `client_secret` (one-time).

## Helpers
- `generateAgentSecret` (line 27) — 32 random bytes hex + SHA-256 hash.
- `auditAgent` (line 39), `emitAgentEvent` (line 55).
- `getAgentByIDOrClientID` (line 530).

## Imports of note
- `internal/storage` — Agent CRUD
- `crypto/rand`, `crypto/sha256`

## Wired by
- `internal/api/router.go:510-518`

## Notes
- Plaintext `client_secret` returned only on create + rotate. All later reads omit the field.
- `RedirectURIs` accepts both `redirect_uris` and the alias `allowed_callback_urls` for compatibility.
