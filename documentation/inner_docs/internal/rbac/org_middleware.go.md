# org_middleware.go

**Path:** `internal/rbac/org_middleware.go`
**Package:** `rbac`
**LOC:** 112
**Tests:** `org_middleware_test.go`

## Purpose
HTTP middleware for org-scoped RBAC enforcement: RequireOrgMembership, RequireOrgPermission. Scopes checks to URL parameter org_id or id.

## Key types / functions
- `RequireOrgMembership()` (line 13) — returns http.Handler middleware that enforces caller is org member
- `RequireOrgPermission()` (line 48) — returns http.Handler middleware that enforces caller has (action, resource) permission in org
- `writeJSONError()` (line 83) — inline JSON error writer (avoids encoding/json import)
- `jsonEscape()` (line 93) — escapes JSON string literals for error messages

## Imports of note
- `github.com/go-chi/chi/v5` — URL parameter extraction
- `internal/api/middleware` — GetUserID from context

## Wired by
- Chi route definitions in `internal/api/org_rbac_handlers.go` (wrapped around handler chains)

## Used by
- Org endpoints (GET /orgs/{org_id}/...) to guard with membership/permission checks

## Notes
- Membership check returns 404 ("not found") if user not member, not 403 (line 30).
- Permission check falls back to "id" param if "org_id" not in URL (line 51) for legacy routes.
- Both checks read userID from context set by auth middleware (line 55).
- 401 Unauthorized if no valid session (line 19).
- ErrNotMember translated to 404 to hide org existence from non-members (line 64).
- All permission check errors (including unexpected) return 403, not 500 (line 68).
- JSON error response inlined to avoid importing encoding/json (line 87).
