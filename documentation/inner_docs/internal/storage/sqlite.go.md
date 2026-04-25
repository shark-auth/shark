# sqlite.go

**Path:** `internal/storage/sqlite.go`
**Package:** `storage`
**LOC:** 1583
**Tests:** `sqlite_test.go`

## Purpose
Master SQLite implementation file: opens the connection, configures pragmas, and implements the user/session/stats/audit/role/permission/SSO/API-key/passkey/magic-link/MFA/migration slice of the `Store` interface. Domain-specific slices (oauth, orgs, webhooks, vault, etc.) live in sibling `*_sqlite.go` files but all attach to the same `*SQLiteStore` receiver.

## Type
- `SQLiteStore struct { db *sql.DB }` (line 14)
- `NewSQLiteStore(dsn string) (*SQLiteStore, error)` (line 20) — opens connection, pings, sets `WAL` + `foreign_keys=ON`. Caps `MaxOpenConns=1` for `:memory:` so all goroutines share the same in-memory DB.
- `DB()` (line 54) returns underlying `*sql.DB`
- `Close()` (line 59)

## Interface methods implemented (selection)
- Users CRUD: `CreateUser` (65), `GetUserByID` (76), `GetUserByEmail` (82), `ListUsers` (88) with multi-filter WHERE builder, `UpdateUser` (158), `DeleteUser` (170)
- `MarkWelcomeEmailSent` (179) — atomic UPDATE with `WHERE welcome_email_sent = 0` guard, returns `sql.ErrNoRows` for idempotency
- Sessions CRUD + `ListActiveSessions` keyset pagination, `DeleteAllActiveSessions`, `DeleteSessionsByUserID`
- Stats: `CountUsers`, `CountUsersCreatedSince`, `CountActiveSessions`, `CountMFAEnabled`, `CountFailedLoginsSince`, `CountExpiringAPIKeys`, `GroupSessionsByAuthMethodSince`, `GroupUsersCreatedByDay`
- Roles + Permissions + RolePermissions + UserRoles full CRUD with batch counters (`BatchCountRolesByPermissionIDs`, `BatchCountUsersByPermissionIDs`)
- SSO Connections + Identities
- API Keys CRUD + `RevokeAPIKey`, `CountActiveAPIKeysByScope`
- Passkey CRUD; MagicLink CRUD; MFA recovery codes CRUD
- Audit logs: `CreateAuditLog`, `GetAuditLogByID`, `QueryAuditLogs` (cursor pagination, multi-filter), `DeleteAuditLogsBefore`
- Auth0 import migrations CRUD
- DevEmails CRUD

## Tables touched
- users, sessions, roles, permissions, role_permissions, user_roles
- sso_connections, sso_identities
- api_keys, audit_logs, migrations
- passkey_credentials, magic_link_tokens, mfa_recovery_codes
- dev_emails

## Imports of note
- `database/sql`, `_ modernc.org/sqlite` (CGO-free driver)

## Used by
- Every package that talks to persistence: `internal/api`, `internal/auth`, `internal/oauth`, `internal/server`, `internal/identity`, `cmd/shark`.
- Wired in `internal/server/build.go` via `storage.NewSQLiteStore` then handed to `api.New`.

## Notes
- WAL mode + `foreign_keys=ON` set via `PRAGMA` after Open since modernc.org/sqlite ignores DSN pragmas.
- `:memory:` special-cases `MaxOpenConns=1` because each connection in modernc.org/sqlite gets a private in-memory database otherwise.
- Helpers: `boolToInt(b)` and `scanUser(s)` / `scanUserFromRows(rows)` patterns are reused across domain files.
- Sibling files extend `*SQLiteStore` (Go method-set across files in the same package) — there is no separate "OauthStore"; everything is methods on `SQLiteStore`.
