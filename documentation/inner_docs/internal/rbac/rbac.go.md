# rbac.go

**Path:** `internal/rbac/rbac.go`
**Package:** `rbac`
**LOC:** 334
**Tests:** `rbac_test.go`

## Purpose
RBAC permission checking and role management: global HasPermission, GetEffectivePermissions (with wildcard matching), org-scoped variants, role seeding.

## Key types / functions
- `RBACManager` (struct, line 21) — wraps storage.Store for role/permission queries
- `NewRBACManager()` (line 26) — construct from store
- `RBACManager.HasPermission()` (line 32) — checks if user has action+resource through any role; supports wildcard (*)
- `RBACManager.GetEffectivePermissions()` (line 51) — resolves all permissions for user via assigned roles, deduplicates by ID
- `RBACManager.SeedDefaultRoles()` (line 78) — creates admin (wildcard) + member roles if missing
- `RBACManager.IsOrgMember()` (line 147) — checks if user has any roles in org
- `RBACManager.HasOrgPermission()` (line 159) — org-scoped permission check; returns ErrNotMember if user not in org
- `RBACManager.GetEffectiveOrgPermissions()` (line 186) — org-scoped version of GetEffectivePermissions

## Imports of note
- `github.com/matoous/go-nanoid/v2` — role/permission ID generation
- `internal/storage` — storage.Store interface, Role, Permission, User types

## Wired by
- `internal/api/rbac_handlers.go` (role/permission CRUD endpoints)
- `internal/rbac/org_middleware.go` (HTTP middleware for permission checks)
- Server initialization for role seeding

## Used by
- org_middleware.RequireOrgPermission for HTTP auth
- Dashboard for role management UI
- Proxy rule engine (ReqPermission, Phase 6.5 placeholder)

## Notes
- Wildcard matching: action="*" matches any action; resource="*" matches any resource (line 39).
- Admin role seeded with wildcard (*/*) permission if missing (line 95).
- Member role created empty (no permissions by default) (line 127).
- Org-scoped checks differentiate from global via GetOrgRolesByUserID (line 160).
- ErrNotMember returned when user has zero roles in org, allowing 404 translation in middleware (line 165).
- Permission deduplication by ID prevents duplicates from overlapping roles (line 57).
