# user_list.go

**Path:** `cmd/shark/cmd/user_list.go`
**Package:** `cmd`

## Purpose
Implements `shark user list` — lists users via GET /api/v1/users with optional `--limit` and `--search` filters.

## Key types / functions
- `userListCmd` — cobra command, tabwriter output, `--json` flag supported.

## Wired by
- `init()` adds to `userCmd` (declared in `user_tier.go`).
