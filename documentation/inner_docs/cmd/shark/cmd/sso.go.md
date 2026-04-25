# sso.go + sso_*.go

**Path:** `cmd/shark/cmd/sso*.go`
**Package:** `cmd`

## Purpose
Implements `shark sso` parent + create/list/show/update/delete subcommands wrapping `/api/v1/sso/connections`.

## Subcommands
- `shark sso list` — GET /api/v1/sso/connections
- `shark sso show <id>` — GET /api/v1/sso/connections/{id}
- `shark sso create --name --type [--domain]` — POST /api/v1/sso/connections
- `shark sso update <id>` — PUT /api/v1/sso/connections/{id}
- `shark sso delete <id>` — DELETE /api/v1/sso/connections/{id}
