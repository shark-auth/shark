# `cmd/shark/cmd/audit.go`

## Purpose

Parent command group for all `shark audit` subcommands. Registers the `audit` noun on the root command and provides the entry point for listing and exporting audit logs via the admin API.

## Command shape

`shark audit <subcommand>`

## Flags

None — flags are defined on individual subcommands.

## API endpoint(s) called

None directly. Delegates to subcommands.

## Example

`shark audit list`
