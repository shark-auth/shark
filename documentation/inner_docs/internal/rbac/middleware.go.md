# middleware.go

**Path:** `internal/rbac/middleware.go`
**Package:** `rbac`
**LOC:** 35
**Tests:** see `rbac_test.go`

## Purpose
HTTP middleware factory that gates a route on a single (action, resource) RBAC permission check. Must run after authentication middleware that sets the user ID on request context.

## Key functions
- `RequirePermission(rbac *RBACManager, action, resource string) func(http.Handler) http.Handler` — returns a chainable middleware. Behaviour:
  - Reads user ID via `mw.GetUserID(r.Context())`.
  - Empty user ID → 401 `{"error":"unauthorized","message":"Authentication required"}`.
  - `rbac.HasPermission(ctx, userID, action, resource)` error → 500 `{"error":"internal_error","message":"Permission check failed"}`.
  - Not allowed → 403 `{"error":"forbidden","message":"Insufficient permissions"}`.
  - Allowed → forwards to `next.ServeHTTP`.

## Imports
- `net/http`
- `mw` alias for `internal/api/middleware` (for `GetUserID`).

## Wired by
- Router setup in `cmd/sharkauth/server.go` / route groups in `internal/api/`. Typical pattern: `r.With(rbac.RequirePermission(rbacMgr, "users", "write")).Post(...)`.

## Used by
- Any admin route requiring permission gating beyond authentication.

## Notes
- Error responses are hand-written JSON strings via `http.Error` (which sends `text/plain` Content-Type). Consider switching to `mw.WriteJSONError` for consistency with the rest of the API.
- Single-permission only — for multi-permission policies callers must compose the middleware or extend `RBACManager`.
