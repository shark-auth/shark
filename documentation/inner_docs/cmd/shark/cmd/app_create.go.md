# app_create.go

**Path:** `cmd/shark/cmd/app_create.go`
**Package:** `cmd`
**LOC:** 214
**Tests:** app_test.go

## Purpose
Implements `shark app create` — generates a client_id and one-time client_secret, validates URL inputs, and inserts a new `storage.Application` row.

## Key types / functions
- `appCreateCmd` (var, line 26):
  - Validates `--name` and all callback/logout/origin URLs.
  - Generates client_id `shark_app_<nanoid21>` and base62-encoded 32-byte secret with sha256 hash + 8-char prefix.
  - Calls `store.CreateApplication`.
- `validateCLIURLs` / `validateCLIURL` (lines 128-150) — rejects `javascript`, `file`, `data`, `vbscript` schemes.
- `generateCLISecret` (func, line 153) — random 32-byte → base62 → secret + sha256 hash + prefix.
- `cliBase62Encode` / `cliIsZero` / `cliDivmod` (lines 170-204) — long-division base62 encoder.

## Imports of note
- `github.com/matoous/go-nanoid/v2`
- `crypto/rand`, `crypto/sha256`
- `internal/config`, `internal/storage`

## Wired by / used by
- Attached to `appCmd` in `init()` line 206.
- Re-uses `appToJSON` from app_list.go.
- `generateCLISecret` is also called by `app_rotate.go`.

## Notes
- Secret is shown exactly once.
- The base62 alphabet matches the auth server's secret format for consistency.
