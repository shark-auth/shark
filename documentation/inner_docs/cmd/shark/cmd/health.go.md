# health.go

**Path:** `cmd/shark/cmd/health.go`
**Package:** `cmd`
**LOC:** 59
**Tests:** none direct (json_output_test.go covers JSON shape)

## Purpose
Implements `shark health` — performs a 5s-timeout GET against `<url>/healthz` and prints OK or detailed error.

## Key types / functions
- `healthCmd` (var, line 14) — cobra command with `RunE`.
- Flags: `--url` (default `http://localhost:8080`), `--json`.
- Decodes the response body and merges its fields into the JSON output payload.

## Imports of note
- `net/http`, `encoding/json`
- Uses package-local `jsonFlag`, `writeJSON`, `writeJSONError`.

## Wired by / used by
- Registered in `cmd/shark/cmd/root.go:94`.

## Notes
- Non-2xx responses are reported with status code and decoded body in details.
- `maybeJSONErr` (defined in app_list.go) wraps the unreachable error path.
