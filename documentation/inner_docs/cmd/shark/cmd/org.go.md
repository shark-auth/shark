# `cmd/shark/cmd/org.go`

## Purpose

Parent command group for all `shark org` subcommands. Registers the `org` noun on the root command and provides the entry point for inspecting and managing organizations via the admin API.

## Command shape

`shark org <subcommand>`

## Flags

None — flags are defined on individual subcommands.

## API endpoint(s) called

None directly. Delegates to subcommands.

## Example

`shark org show org_abc123`
