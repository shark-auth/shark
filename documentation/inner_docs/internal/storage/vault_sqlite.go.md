# vault_sqlite.go

**Path:** `internal/storage/vault_sqlite.go`
**Package:** `storage`
**LOC:** 481
**Tests:** `vault_sqlite_test.go`

## Purpose
SQLite implementation of the Token Vault — provider CRUD plus per-user connection CRUD with encrypted-token round-tripping.

## Interface methods implemented
### Vault providers
- `CreateVaultProvider` (13) — JSON-encodes scopes; NULL-safes `icon_url`
- `GetVaultProviderByID` (36), `GetVaultProviderByName` (43)
- `ListVaultProviders` (50) with `activeOnly` filter
- `UpdateVaultProvider`, `DeleteVaultProvider`

### Vault connections
- `CreateVaultConnection`, `GetVaultConnectionByID`
- `GetVaultConnection(providerID, userID)` — primary lookup for "do I already have a connection?"
- `ListVaultConnectionsByUserID`, `ListVaultConnectionsByProviderID`, `ListAllVaultConnections`
- `UpdateVaultConnection` (full) and `UpdateVaultConnectionTokens(id, accessEnc, refreshEnc, expiresAt)` for refresh-only writes
- `MarkVaultConnectionNeedsReauth(id, needs bool)` — flips the `needs_reauth` flag when a refresh fails
- `DeleteVaultConnection`

## Tables touched
- vault_providers
- vault_connections

## Imports of note
- `database/sql`, `encoding/json`, `time`

## Used by
- `internal/api/vault.go` — admin provider CRUD + user-facing connection list
- `internal/vault/refresher.go` — background refresh worker calls `UpdateVaultConnectionTokens` and `MarkVaultConnectionNeedsReauth`
- `internal/vault/access.go` — on-demand access token retrieval (decrypts at the consumer)

## Notes
- All token + secret columns store the already-encrypted ciphertext; this file does no crypto.
- `UpdateVaultConnectionTokens` is a narrower UPDATE than full `UpdateVaultConnection` so the refresh worker can't accidentally clobber `Scopes`/`Metadata`.
- Scope and Metadata fields JSON-encoded at the boundary (consistent with rest of storage).
