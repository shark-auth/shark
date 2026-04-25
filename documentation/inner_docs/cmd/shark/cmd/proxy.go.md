# proxy.go

**Path:** `cmd/shark/cmd/proxy.go`
**Package:** `cmd`
**LOC:** 56
**Tests:** proxy_test.go

## Purpose
Originally a deprecation stub for the legacy standalone `shark proxy` process. The deprecation message and `proxyCmd` variable live here; `proxy_admin.go` repurposes the same `proxyCmd` into a real subcommand tree by clearing the stub's `RunE` in its own `init()`.

## Key types / functions
- `proxyDeprecationMessage` (const, line 13) — kept exported-via-package so tests can match the literal.
- `proxyCmd` (var, line 36) — registered on `root` in `init()`. Original `RunE` writes the deprecation message to stderr and `os.Exit(2)`.

## Imports of note
- `github.com/spf13/cobra`

## Wired by / used by
- Registered on `root` at line 51.
- Mutated in `proxy_admin.go:init` (replaces RunE, attaches lifecycle + rules subcommands).

## Notes
- Init file order matters: this file's init runs first to create `proxyCmd`; proxy_admin.go's init then overwrites RunE/Short/Long and adds children.
- The deprecation message itself is no longer reachable in v1.5 because RunE is cleared.
