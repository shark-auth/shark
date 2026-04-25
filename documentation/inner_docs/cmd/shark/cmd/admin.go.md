# `cmd/shark/cmd/admin.go`

## Purpose

Parent command group for all `shark admin` subcommands. Registers the `admin` noun on the root command and provides the entry point for administrative operations such as config dump and runtime management.

## Command shape

`shark admin <subcommand>`

## Flags

None — flags are defined on individual subcommands.

## API endpoint(s) called

None directly. Delegates to subcommands.

## Example

`shark admin config dump`
