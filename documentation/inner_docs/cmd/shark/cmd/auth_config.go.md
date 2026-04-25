# `cmd/shark/cmd/auth_config.go`

## Purpose

Parent command group for all `shark auth` subcommands. Registers the `auth` noun on the root command and provides the entry point for inspecting and managing authentication configuration via the admin API.

## Command shape

`shark auth <subcommand>`

## Flags

None — flags are defined on individual subcommands.

## API endpoint(s) called

None directly. Delegates to subcommands.

## Example

`shark auth config show`
