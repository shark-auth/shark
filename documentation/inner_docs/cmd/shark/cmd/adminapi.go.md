# adminapi.go

**Path:** `cmd/shark/cmd/adminapi.go`
**Package:** `cmd`
**LOC:** 151
**Tests:** none direct (covered by proxy_admin_test.go etc)

## Purpose
Shared HTTP client + helpers for every CLI subcommand that talks to the admin HTTP API. Resolves URL/token from flags or env, performs requests, decodes JSON, extracts API errors.

## Key types / functions
- `adminURLFlag`, `adminTokenFlag`, `adminClient` (vars, lines 24-31).
- `resolveAdminURL` (func, line 34) — `--url` > `SHARK_URL` > default `http://localhost:8080`.
- `resolveAdminToken` (func, line 46) — `--token` > `SHARK_ADMIN_TOKEN`; errors if neither.
- `adminDo` (func, line 60) — JSON request/response with status code + decoded body.
- `adminDoRaw` (func, line 99) — raw bytes (used by paywall HTML).
- `apiError` (func, line 122) — extracts human-readable message from various error envelopes.
- `init` (line 147) — registers persistent `--url` and `--token` on root.

## Imports of note
- `net/http`, `encoding/json`, `bytes`, `time`.

## Wired by / used by
- Used by every admin-API subcommand: `proxy_admin.go`, `proxy.go` lifecycle wrappers, `branding.go`, `user_tier.go`, `agent_register.go`, `whoami.go`, `paywall.go`.

## Notes
- 10s default HTTP timeout.
- Trailing slashes stripped from URLs.
