# agents.go

**Path:** `internal/storage/agents.go`
**Package:** `storage`
**LOC:** 39
**Tests:** none direct (types only); `agents_sqlite_test.go` exercises the impl.

## Purpose
Type-only file: declares `Agent` (OAuth 2.1 client with agent identity) and `ListAgentsOpts` (query options for `ListAgents`).

## Types defined
- `Agent` (line 6) — OAuth 2.1 client metadata. Fields cover client credentials (`ClientID`, `ClientSecretHash`), client type (`confidential`/`public`), token endpoint auth method (`client_secret_basic`/`_post`/`private_key_jwt`/`none`), JWKS, redirect URIs, grant + response types, scopes, lifetimes, branding (logo + homepage), F4.3 secret-rotation fields (`OldSecretHash`, `OldSecretExpiresAt`).
- `ListAgentsOpts` (line 34) — pagination + search + active-only filter.

## Used by
- `internal/storage/agents_sqlite.go` — implementation.
- `internal/api/agents.go` — admin CRUD handlers.
- `internal/oauth/store.go` — fosite `GetClient` adapter reads agents.
- `internal/oauth/handlers.go` — token endpoint client auth.

## Notes
- `ClientSecretHash` and `OldSecretHash` are `json:"-"` so they never serialize.
- Slice fields (`RedirectURIs`, `GrantTypes`, etc.) and `Metadata` are JSON-encoded at the SQLite boundary in `agents_sqlite.go`.
- `OldSecretExpiresAt` powers the 1-hour DCR rotation grace window per F4.3.
