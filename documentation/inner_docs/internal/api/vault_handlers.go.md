# vault_handlers.go

**Path:** `internal/api/vault_handlers.go`
**Package:** `api`
**LOC:** 879
**Tests:** likely integration-tested (largest handler file outside admin_system)

## Purpose
"Vault" feature — third-party OAuth credential storage so a user can "Connect to GitHub" once and have SharkAuth refresh + return tokens to apps on demand. Three concerns: provider CRUD (admin), connect+callback OAuth flow (user-facing), and token retrieval (M2M for apps via API key).

## Handlers exposed
**Provider CRUD (admin)**
- `handleCreateVaultProvider` (line 150) — POST. Accepts a template key (fills auth/token URLs from `vault.Template`) or explicit definition. Always requires plaintext `client_id` + `client_secret`.
- `handleListVaultProviders` (line 260), `handleGetVaultProvider` (line 278)
- `handleUpdateVaultProvider` (line 292), `handleDeleteVaultProvider` (line 364)
- `handleListVaultTemplates` (line 386) — GET catalog of `vault.Templates()`.

**Connect / callback (user)**
- `handleVaultConnectStart` (line 422) — GET `/vault/connect/{provider}`. Generates state + connection-cookie, redirects to provider authorize URL.
- `handleVaultCallback` (line 472) — GET `/vault/callback/{provider}`. Exchanges code, encrypts + persists tokens.
- `handleListVaultConnections` (line 711) — GET. Per-user list with `needs_reauth` flag.
- `handleDeleteVaultConnection` (line 755) — DELETE. Per-user.

**Token retrieval (M2M)**
- `handleVaultGetToken` (line 576) — GET `/vault/{provider}/token`. Refreshes if needed, returns plaintext access token.

**Admin cross-user**
- `handleAdminListVaultConnections` (line 804), `handleAdminDeleteVaultConnection` (line 855)

## Key types
- `vaultProviderResponse` (line 42) — never includes encrypted client_secret.
- `vaultConnectionResponse` (line 81) — never includes token material; carries `needs_reauth` + provider display fields.

## Helpers
- `sanitizeVaultProvider` (line 56), `isHTTPSURL` (line 98)
- `auditVault` (line 117) — uniform audit wrapper. Audit constants `auditVault*` (lines 25-32).
- `isDuplicateErr` (line 248), `vaultStateValue`/`parseVaultStateValue` (lines 396, 398)
- `vaultRedirectURI` (line 406), `scopeContains` (line 698)

## Constants
- `vaultStateCookieName = "shark_vault_state"`, `vaultStateTTL = 5*time.Minute`

## Imports of note
- `internal/vault` — `Template`, `ApplyTemplate`
- `internal/storage` — `VaultProvider`, `VaultConnection`
- `internal/api/middleware` — user context
- `crypto/subtle` — constant-time state compare

## Wired by
- `internal/api/router.go:548-562` (provider CRUD + connect/callback/connections), `:569` (M2M token), `:599-600` (admin)

## Notes
- Provider client_secret is encrypted at rest; rotation goes through dedicated PATCH field, never round-tripped via GET.
- `needs_reauth=true` when the refresh token is missing/expired and the connection can't be silently renewed.
- HTTPS URL rule is enforced on auth_url + token_url; localhost http is allowed for dev.
