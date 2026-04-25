# vault.go

**Path:** `internal/storage/vault.go`
**Package:** `storage`
**LOC:** 43
**Tests:** indirectly via `vault_sqlite_test.go`.

## Purpose
Type-only file: declares the Token Vault entities — admin-configured third-party OAuth providers and per-user encrypted-token connections to those providers (Google Calendar, Slack, GitHub, etc.).

## Types defined
- `VaultProvider` (line 11) — admin-configured provider. `ClientSecretEnc` is stored encrypted (`enc::<b64>` prefix) via `FieldEncryptor`; handlers decrypt before OAuth flows. Storage layer never touches crypto.
- `VaultConnection` (line 29) — single user's link to a provider. Holds encrypted access + refresh tokens, scopes granted, optional expiry, and a `NeedsReauth` flag set hot when refresh fails.

## Used by
- `internal/storage/vault_sqlite.go` — implementation.
- `internal/api/vault.go` admin + user-facing CRUD.
- `internal/vault` — token refresh worker, on-demand access-token resolution.

## Notes
- `ClientSecretEnc`, `AccessTokenEnc`, `RefreshTokenEnc` all `json:"-"` so they never serialize.
- `RefreshTokenEnc` may be empty when a provider issues no refresh token (some providers issue access-only).
- `LastRefreshedAt` lets the dashboard surface freshness without inferring from `UpdatedAt`.
