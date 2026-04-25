# `cmd/shark/cmd/debug.go`

## Purpose

Parent command group for all `shark debug` subcommands. Registers the `debug` noun on the root command and provides the entry point for debugging utilities such as token inspection and session analysis.

## Command shape

`shark debug <subcommand>`

## Flags

None — flags are defined on individual subcommands.

## API endpoint(s) called

None directly. Delegates to subcommands.

## Example

`shark debug token --token <jwt>`
