# user.go

**Path:** `internal/user/user.go`  
**Package:** `user`  
**LOC:** 83  
**Tests:** `user_test.go`

## Purpose
User CRUD service backed by storage.Store. Factory for generating user IDs, creating/reading/updating/deleting user records.

## Key types / functions
- `Service` (struct, line 20) — wraps storage.Store
- `NewService(store)` (func, line 25) — constructor
- `NewID()` (func, line 14) — generates new user ID (usr_* prefix + nanoid)
- `Create(ctx, email, passwordHash, name)` (func, line 30) — creates user with RFC3339 timestamps, argon2id hash type
- `GetByID(ctx, id)` (func, line 60) — lookup by user ID
- `GetByEmail(ctx, email)` (func, line 65) — lookup by email
- `List(ctx, opts)` (func, line 70) — paginated user list
- `Update(ctx, u)` (func, line 75) — updates user, refreshes UpdatedAt timestamp
- `Delete(ctx, id)` (func, line 81) — removes user

## Imports of note
- `gonanoid` — nanoid ID generation
- `time` — RFC3339 formatting
- `internal/storage` — Store interface, User struct

## Wired by
- Auth handlers call Service methods for user creation (email, password, OAuth flows)
- Admin API uses List/GetByEmail for user management endpoints
- Signup flows call Create() + Update() for progressive fields

## Notes
- ID format: `usr_` + nanoid (21 chars typically)
- Default hash type: `argon2id` when password provided; "" when passwordless
- Metadata initialized as empty JSON object "{}"
- Timestamps: RFC3339 UTC format
- Name optional (pointer); nil if not provided
- All methods delegate to storage.Store; Service is a thin wrapper for CRUD

