# server.go

**Path:** `internal/testutil/server.go`
**Package:** `testutil`
**LOC:** 646
**Tests:** none (test helper).

## Purpose
Spins up a fully wired `httptest.Server` with the real `internal/api` router, a memory email sender, in-memory DB with migrations, an admin API key, and a default seeded application. Provides an HTTP client with a cookie jar plus a battery of helpers (`PostJSON`, `Authenticated`, etc.) so handler tests look like a few lines of integration. Test helper — **not for production runtime.**

## Type
- `TestServer` (line 29) wraps:
  - `*httptest.Server` + `*http.Client` (cookie jar)
  - `storage.Store`, `*config.Config`, `*api.Server`
  - `*MemoryEmailSender`
  - `AdminKey` — full `sk_live_...` for Bearer auth

## Functions (selection)
- `seedTestDefaultApp(t, store, cfg)` (line 43) — idempotent default-application insert, seeds magic-link + social redirect URLs into the allowlist so production-style validation works
- `NewTestServer(t)` and friends — composition of `NewTestDB`, `TestConfig`, `api.NewServer`, `httptest.NewServer`
- HTTP helpers: typed JSON body marshaling, automatic cookie carry-over, admin key Bearer injection
- Auth helpers for logging in test users and asserting session state

## Imports of note
- `net/http/httptest`, `net/http/cookiejar`
- `internal/api`, `internal/auth`, `internal/auth/jwt`, `internal/config`, `internal/email`, `internal/storage`

## Used by
- Every handler test in `internal/api`, `internal/oauth`, `internal/auth`

## Notes
- Test helper — **not for production runtime.**
- Real chi router — full middleware chain — so tests are integration-flavored, not unit.
- `AdminKey` is generated per-test, so concurrent tests don't share auth state.
- Default app seeding is idempotent so tests can recreate the helper without conflicts.
