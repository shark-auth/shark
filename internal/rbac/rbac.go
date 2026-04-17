package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// Sentinel errors for org-scoped RBAC.
var (
	ErrNotMember   = errors.New("user is not a member of organization")
	ErrBuiltinRole = errors.New("cannot delete builtin role")
)

// RBACManager provides permission checking and role management.
type RBACManager struct {
	store storage.Store
}

// NewRBACManager creates a new RBACManager with the given store.
func NewRBACManager(store storage.Store) *RBACManager {
	return &RBACManager{store: store}
}

// HasPermission checks if a user has a specific action+resource permission
// through any of their assigned roles. Supports wildcard (*) matching.
func (r *RBACManager) HasPermission(ctx context.Context, userID, action, resource string) (bool, error) {
	perms, err := r.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, p := range perms {
		actionMatch := p.Action == "*" || p.Action == action
		resourceMatch := p.Resource == "*" || p.Resource == resource
		if actionMatch && resourceMatch {
			return true, nil
		}
	}

	return false, nil
}

// GetEffectivePermissions resolves all permissions for a user through their roles.
// Permissions are de-duplicated by ID.
func (r *RBACManager) GetEffectivePermissions(ctx context.Context, userID string) ([]*storage.Permission, error) {
	roles, err := r.store.GetRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var result []*storage.Permission

	for _, role := range roles {
		perms, err := r.store.GetPermissionsByRoleID(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range perms {
			if !seen[p.ID] {
				seen[p.ID] = true
				result = append(result, p)
			}
		}
	}

	return result, nil
}

// SeedDefaultRoles creates the admin and member roles with default permissions
// if they don't already exist. The admin role gets wildcard (*/*) permission.
func (r *RBACManager) SeedDefaultRoles(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Seed admin role
	_, err := r.store.GetRoleByName(ctx, "admin")
	if errors.Is(err, sql.ErrNoRows) {
		id, _ := gonanoid.New()
		adminRole := &storage.Role{
			ID:          "role_" + id,
			Name:        "admin",
			Description: "Full administrative access",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := r.store.CreateRole(ctx, adminRole); err != nil {
			return err
		}

		// Create wildcard permission for admin
		permID, _ := gonanoid.New()
		wildcardPerm := &storage.Permission{
			ID:        "perm_" + permID,
			Action:    "*",
			Resource:  "*",
			CreatedAt: now,
		}
		// Check if wildcard perm already exists
		existing, err := r.store.GetPermissionByActionResource(ctx, "*", "*")
		if errors.Is(err, sql.ErrNoRows) {
			if err := r.store.CreatePermission(ctx, wildcardPerm); err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			wildcardPerm = existing
		}

		if err := r.store.AttachPermissionToRole(ctx, adminRole.ID, wildcardPerm.ID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Seed member role
	_, err = r.store.GetRoleByName(ctx, "member")
	if errors.Is(err, sql.ErrNoRows) {
		id, _ := gonanoid.New()
		memberRole := &storage.Role{
			ID:          "role_" + id,
			Name:        "member",
			Description: "Standard member access",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := r.store.CreateRole(ctx, memberRole); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// --- Org-scoped RBAC ---

// HasOrgPermission checks if a user has a specific (action, resource) permission
// in the given org through any of their org roles. Wildcards are supported:
// action="*" matches any action; resource="*" matches any resource.
// Returns ErrNotMember if the user has no roles in the org at all.
func (r *RBACManager) HasOrgPermission(ctx context.Context, userID, orgID, action, resource string) (bool, error) {
	roles, err := r.store.GetOrgRolesByUserID(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	if len(roles) == 0 {
		return false, ErrNotMember
	}

	for _, role := range roles {
		perms, err := r.store.GetOrgRolePermissions(ctx, role.ID)
		if err != nil {
			return false, err
		}
		for _, p := range perms {
			actionMatch := p.Action == "*" || p.Action == action
			resourceMatch := p.Resource == "*" || p.Resource == resource
			if actionMatch && resourceMatch {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetEffectiveOrgPermissions returns the deduplicated set of permissions a user
// has in the given org through all of their org roles.
func (r *RBACManager) GetEffectiveOrgPermissions(ctx context.Context, userID, orgID string) ([]storage.Permission, error) {
	roles, err := r.store.GetOrgRolesByUserID(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var result []storage.Permission

	for _, role := range roles {
		perms, err := r.store.GetOrgRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range perms {
			key := p.Action + ":" + p.Resource
			if !seen[key] {
				seen[key] = true
				result = append(result, p)
			}
		}
	}
	return result, nil
}

// CreateOrgRole creates a new custom (non-builtin) org role with a generated ID.
func (r *RBACManager) CreateOrgRole(ctx context.Context, orgID, name, description string) (*storage.OrgRole, error) {
	id, err := gonanoid.New()
	if err != nil {
		return nil, fmt.Errorf("generating org role id: %w", err)
	}
	roleID := "orgrole_" + id
	if err := r.store.CreateOrgRole(ctx, orgID, roleID, name, description, false); err != nil {
		return nil, err
	}
	return r.store.GetOrgRoleByID(ctx, roleID)
}

// DeleteOrgRole deletes an org role. Returns ErrBuiltinRole if the role is builtin.
func (r *RBACManager) DeleteOrgRole(ctx context.Context, orgID, roleID string) error {
	role, err := r.store.GetOrgRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsBuiltin {
		return ErrBuiltinRole
	}
	return r.store.DeleteOrgRole(ctx, roleID)
}

// GrantOrgRole assigns an org role to a user in the given org.
func (r *RBACManager) GrantOrgRole(ctx context.Context, orgID, userID, roleID, grantedBy string) error {
	return r.store.GrantOrgRole(ctx, orgID, userID, roleID, grantedBy)
}

// RevokeOrgRole removes an org role from a user in the given org.
func (r *RBACManager) RevokeOrgRole(ctx context.Context, orgID, userID, roleID string) error {
	return r.store.RevokeOrgRole(ctx, orgID, userID, roleID)
}

// AttachOrgPermission attaches an (action, resource) permission to an org role.
func (r *RBACManager) AttachOrgPermission(ctx context.Context, orgRoleID, action, resource string) error {
	return r.store.AttachOrgPermission(ctx, orgRoleID, action, resource)
}

// DetachOrgPermission removes an (action, resource) permission from an org role.
func (r *RBACManager) DetachOrgPermission(ctx context.Context, orgRoleID, action, resource string) error {
	return r.store.DetachOrgPermission(ctx, orgRoleID, action, resource)
}

// SeedOrgRoles creates the three builtin org roles (owner, admin, member) with
// their canonical permissions. The operation is idempotent: duplicate inserts
// are silently ignored via INSERT OR IGNORE / UNIQUE constraint catch.
//
// Roles seeded:
//
//	owner  — (*,*) full wildcard
//	admin  — members:*, org:update, roles:create, roles:assign, roles:revoke, webhooks:manage
//	member — members:read, org:read
func (r *RBACManager) SeedOrgRoles(ctx context.Context, orgID string) error {
	type builtinRole struct {
		name        string
		description string
		permissions [][2]string // {action, resource}
	}

	roles := []builtinRole{
		{
			name:        "owner",
			description: "Full administrative access",
			permissions: [][2]string{{"*", "*"}},
		},
		{
			name:        "admin",
			description: "Administrative access with member and role management",
			// Permission strings use action:resource notation, e.g. members:read
			// maps to action="members", resource="read".
			// members:* is stored as action="members", resource="*" (wildcard).
			permissions: [][2]string{
				{"members", "*"}, // wildcard — covers read, invite, remove, update_role
				{"org", "update"},
				{"roles", "create"},
				{"roles", "assign"},
				{"roles", "revoke"},
				{"webhooks", "manage"},
			},
		},
		{
			name:        "member",
			description: "Standard member access",
			permissions: [][2]string{
				{"members", "read"},
				{"org", "read"},
			},
		},
	}

	for _, br := range roles {
		// Attempt to get existing role; if not found, create it.
		existing, err := r.store.GetOrgRoleByName(ctx, orgID, br.name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("checking org role %q: %w", br.name, err)
		}

		var roleID string
		if existing != nil {
			roleID = existing.ID
		} else {
			nid, err := gonanoid.New()
			if err != nil {
				return fmt.Errorf("generating id for org role %q: %w", br.name, err)
			}
			roleID = "orgrole_" + nid
			if err := r.store.CreateOrgRole(ctx, orgID, roleID, br.name, br.description, true); err != nil {
				return fmt.Errorf("creating org role %q: %w", br.name, err)
			}
		}

		// Attach permissions (idempotent via INSERT OR IGNORE in AttachOrgPermission).
		for _, perm := range br.permissions {
			if err := r.store.AttachOrgPermission(ctx, roleID, perm[0], perm[1]); err != nil {
				return fmt.Errorf("attaching permission %s:%s to role %q: %w", perm[0], perm[1], br.name, err)
			}
		}
	}

	return nil
}

