# user_create.go

**Path:** `cmd/shark/cmd/user_create.go`
**Package:** `cmd`

## Purpose
Implements `shark user create` — creates a user via POST /api/v1/admin/users. `--email` is required.

## Key types / functions
- `userCreateCmd` — cobra command, `--json` flag supported.
