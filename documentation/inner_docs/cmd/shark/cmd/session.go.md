# session.go + session_*.go

**Path:** `cmd/shark/cmd/session*.go`
**Package:** `cmd`

## Purpose
Implements `shark session` parent + list/show/revoke subcommands wrapping admin sessions API.

## Subcommands
- `shark session list [--user <id>] [--limit N]` — GET /api/v1/admin/sessions
- `shark session show <id>` — GET /api/v1/admin/sessions?session_id={id}
- `shark session revoke <id>` — DELETE /api/v1/admin/sessions/{id}
