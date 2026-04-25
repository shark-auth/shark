# user_tier.go

**Path:** `cmd/shark/cmd/user_tier.go`
**Package:** `cmd`
**LOC:** 114
**Tests:** user_tier_test.go

## Purpose
Implements `shark user tier <user-id-or-email> <tier>` — sets a user's tier (free|pro) via `PATCH /api/v1/admin/users/{id}/tier`, resolving an email-shaped ref to an ID first.

## Key types / functions
- `userCmd` (var, line 14) — parent.
- `userTierCmd` (var, line 19):
  - Validates tier ∈ {free, pro}.
  - Calls `resolveUserID` for email refs.
  - PATCHes the tier endpoint, maps 404 → not_found.
- `resolveUserID` (func, line 66) — heuristic: contains `@` ⇒ search by email via `GET /api/v1/users?search=...`; picks exact match else first result.

## Imports of note
- `encoding/json`, `net/url`, `strings`
- Uses `adminDo`, `apiError`, `maybeJSONErr`.

## Wired by / used by
- Registered on `root` in `init()` at line 110.

## Notes
- Lane E, milestone E4.
- The email→ID lookup uses `limit=5` and prefers exact (case-insensitive) email match.
