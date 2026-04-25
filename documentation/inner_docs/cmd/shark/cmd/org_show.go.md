# `cmd/shark/cmd/org_show.go`

## Purpose

Implements `shark org show <id>` — fetches a single organization by ID and displays its id, name, slug, and creation date. Returns a structured 404 error if the org does not exist.

## Command shape

`shark org show <id> [--json]`

## Flags

- `--json` — emit raw JSON output

## API endpoint(s) called

- `GET /api/v1/admin/organizations/{id}` — retrieve a single organization

## Example

`shark org show org_abc123`
