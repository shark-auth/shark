# `cmd/shark/cmd/consent.go`

## Purpose

Parent command group for all `shark consents` subcommands. Registers the `consents` noun on the root command and provides the entry point for listing and revoking OAuth consents via the admin API.

## Command shape

`shark consents <subcommand>`

## Flags

None — flags are defined on individual subcommands.

## API endpoint(s) called

None directly. Delegates to subcommands.

## Example

`shark consents list`
