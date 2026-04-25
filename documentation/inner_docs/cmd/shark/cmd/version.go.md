# version.go

**Path:** `cmd/shark/cmd/version.go`
**Package:** `cmd`
**LOC:** 30
**Tests:** none

## Purpose
Implements `shark version` — prints the build-injected version, falling back to `debug.ReadBuildInfo` when run via `go install`.

## Key types / functions
- `version` (var, line 12) — overridden via `-ldflags "-X ...cmd.version=vX.Y.Z"`.
- `versionCmd` (var, line 14) — cobra command.
- `resolveVersion` (func, line 22) — resolution with `(devel)`/`dev` fallbacks.

## Imports of note
- `runtime/debug` for module version embedding.

## Wired by / used by
- Registered in `cmd/shark/cmd/root.go:95`.

## Notes
- Returns `"dev"` when neither override nor build info gives a meaningful value.
