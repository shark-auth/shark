# vault.go

**Path:** `internal/vault/vault.go`  
**Package:** `vault`  
**LOC:** 480  
**Tests:** `vault_test.go`

## Purpose
Token Vault: OAuth 2.0 credential and access-token storage with automatic refresh. Manages third-party provider configs + user connections, encrypts credentials, refreshes expired tokens on retrieval.

## Key types / functions
- `Manager` (struct, line 57) — orchestrates vault ops; holds store, encryptor, clock
- `NewManager(store, encryptor)` (func, line 64) — constructor with real wall-clock
- `NewManagerWithClock(store, encryptor, now)` (func, line 74) — test variant with injectable clock
- Sentinel errors: `ErrNeedsReauth`, `ErrNoRefreshToken`, `ErrConnectionNotFound`, `ErrProviderNotFound`
- `Manager.CreateProvider(ctx, provider, clientSecretPlain)` (func, line 88) — encrypt and persist provider config
- `Manager.UpdateProviderSecret(ctx, providerID, clientSecretPlain)` (func, line 132) — rotate provider secret
- `Manager.GetFreshToken(ctx, connectionID)` (func) — retrieve token, auto-refresh if expired
- `Manager.StartAuthorizeFlow(ctx, userID, providerID, scopes)` (func) — build oauth2.Config, return auth URL
- `Manager.ExchangeCode(ctx, connectionID, code)` (func) — code → access token via provider
- `Manager.Disconnect(ctx, connectionID)` (func) — revoke and delete connection

## Imports of note
- `crypto/rand`, `encoding/hex` — random ID generation
- `golang.org/x/oauth2` — OAuth 2.0 client library
- `internal/auth` — FieldEncryptor for credential encryption
- `internal/storage` — VaultProvider, Connection types

## Wired by
- `server.Build()` passes Manager to API handlers
- Agents + dashboards request fresh tokens via GetFreshToken()
- Admin API manages providers (create, rotate secret, list)

## Notes
- Vault ≠ OAuth AS; Vault stores 3rd-party credentials (Google, GitHub, Slack) for agents to use on user's behalf
- Credentials encrypted at rest via AES-256-GCM (internal/auth.FieldEncryptor)
- expiryLeeway = 30s; tokens refreshed slightly early to avoid mid-request expiry
- Connection refresh: Vault calls upstream provider's token endpoint if refresh token present
- Handles provider-specific errors: refresh token rejected, no refresh token, network failures
- Clock-injectable for deterministic test timing of refresh logic

