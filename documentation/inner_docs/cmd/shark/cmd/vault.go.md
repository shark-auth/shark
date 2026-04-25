# `cmd/shark/cmd/vault.go`

## Purpose

Parent command group for all `shark vault` subcommands. Registers the `vault` noun on the root command and the `vault provider` sub-group, providing the entry point for inspecting vault provider configurations via the admin API.

## Command shape

`shark vault <subcommand>`

## Flags

None — flags are defined on individual subcommands.

## API endpoint(s) called

None directly. Delegates to subcommands.

## Example

`shark vault provider show default`
