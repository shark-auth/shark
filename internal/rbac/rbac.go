package rbac

import (
	"context"
	"database/sql"
	"errors"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
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
