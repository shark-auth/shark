# dcr.go

**Path:** `internal/oauth/dcr.go`
**Package:** `oauth`
**LOC:** 666
**Tests:** `dcr_test.go`, `dcr_rotation_test.go`

## Purpose
Dynamic Client Registration (RFC 7591) + Management protocol (RFC 7592) including secret rotation with a 1-hour grace window.

## RFCs implemented
- RFC 7591 Dynamic Client Registration
- RFC 7592 DCR Management Protocol (GET/PUT/DELETE on `/oauth/register/{client_id}`)

## Key types / functions
- `secretRotationGrace` (const, line 24) — 1 hour window where both old + new secrets are accepted.
- `allowedGrantTypes` (var, line 27) — whitelist for DCR-issued clients.
- `dcrMetadata` / `dcrResponse` / `dcrError` (structs, line 35-74) — RFC 7591 wire shapes.
- `validateDCRMetadata` (func, line 91) — required fields, default grant_types/response_types, scheme-restricted redirect URIs.
- `validateRedirectURI` (func, line 139) — only https, or http://localhost|127.0.0.1|::1.
- `generateDCRToken` (func, line 160) — 32-byte hex registration access token + sha256 hash.
- `extractRegistrationToken` / `verifyRegistrationToken` (line 172 / 182) — Bearer auth with constant-time compare for RFC 7592.
- `HandleRegister`, `HandleGetClient`, `HandleUpdateClient`, `HandleDeleteClient`, `HandleRotateSecret` (later) — endpoints.

## Imports of note
- `github.com/go-chi/chi/v5` for URL params
- `crypto/subtle` for constant-time token compare
- `internal/storage` (oauth_dcr_clients table)

## Wired by
- `internal/server/server.go` — mounts `/oauth/register` (POST) and `/oauth/register/{client_id}` (GET/PUT/DELETE/POST rotate).

## Used by
- MCP clients, Claude Desktop, dynamic IDE clients.

## Notes
- Old secret stored in `old_secret_hash` + `old_secret_expires_at`; `agentToClient` (store.go:99) injects it into fosite's `RotatedSecrets` while live.
- DCR clients backed by the same `agents` table as first-party agents — uniform downstream auth.
