package storage

import (
	"context"
	"time"
)

// --- Org RBAC ---
//
// These methods operate on parallel tables (org_roles, org_role_permissions,
// org_user_roles). The global RBAC tables (roles, permissions, role_permissions,
// user_roles) are not touched here.

func (s *SQLiteStore) CreateOrgRole(ctx context.Context, orgID, id, name, description string, isBuiltin bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO org_roles (id, organization_id, name, description, is_builtin, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, orgID, name, description, boolToInt(isBuiltin), now, now,
	)
	return err
}

func (s *SQLiteStore) GetOrgRoleByID(ctx context.Context, roleID string) (*OrgRole, error) {
	var r OrgRole
	var isBuiltin int
	var description *string
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, organization_id, name, description, is_builtin, created_at, updated_at
		 FROM org_roles WHERE id = ?`, roleID,
	).Scan(&r.ID, &r.OrganizationID, &r.Name, &description, &isBuiltin, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if description != nil {
		r.Description = *description
	}
	r.IsBuiltin = isBuiltin != 0
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &r, nil
}

func (s *SQLiteStore) GetOrgRolesByOrgID(ctx context.Context, orgID string) ([]*OrgRole, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, organization_id, name, description, is_builtin, created_at, updated_at
		 FROM org_roles WHERE organization_id = ? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*OrgRole
	for rows.Next() {
		r, err := scanOrgRole(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetOrgRolesByUserID(ctx context.Context, userID, orgID string) ([]*OrgRole, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT r.id, r.organization_id, r.name, r.description, r.is_builtin, r.created_at, r.updated_at
		 FROM org_roles r
		 INNER JOIN org_user_roles ur ON ur.org_role_id = r.id
		 WHERE ur.user_id = ? AND ur.organization_id = ?
		 ORDER BY r.name`, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*OrgRole
	for rows.Next() {
		r, err := scanOrgRole(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetOrgRoleByName(ctx context.Context, orgID, name string) (*OrgRole, error) {
	var r OrgRole
	var isBuiltin int
	var description *string
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, organization_id, name, description, is_builtin, created_at, updated_at
		 FROM org_roles WHERE organization_id = ? AND name = ?`, orgID, name,
	).Scan(&r.ID, &r.OrganizationID, &r.Name, &description, &isBuiltin, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if description != nil {
		r.Description = *description
	}
	r.IsBuiltin = isBuiltin != 0
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &r, nil
}

func (s *SQLiteStore) UpdateOrgRole(ctx context.Context, roleID, name, description string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`UPDATE org_roles SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, now, roleID,
	)
	return err
}

func (s *SQLiteStore) DeleteOrgRole(ctx context.Context, roleID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM org_roles WHERE id = ?`, roleID)
	return err
}

func (s *SQLiteStore) AttachOrgPermission(ctx context.Context, orgRoleID, action, resource string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO org_role_permissions (org_role_id, action, resource) VALUES (?, ?, ?)`,
		orgRoleID, action, resource,
	)
	return err
}

func (s *SQLiteStore) DetachOrgPermission(ctx context.Context, orgRoleID, action, resource string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM org_role_permissions WHERE org_role_id = ? AND action = ? AND resource = ?`,
		orgRoleID, action, resource,
	)
	return err
}

func (s *SQLiteStore) GetOrgRolePermissions(ctx context.Context, orgRoleID string) ([]Permission, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT action, resource FROM org_role_permissions WHERE org_role_id = ? ORDER BY resource, action`,
		orgRoleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Action, &p.Resource); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (s *SQLiteStore) GrantOrgRole(ctx context.Context, orgID, userID, orgRoleID, grantedBy string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO org_user_roles (organization_id, user_id, org_role_id, granted_by)
		 VALUES (?, ?, ?, ?)`,
		orgID, userID, orgRoleID, grantedBy,
	)
	return err
}

func (s *SQLiteStore) RevokeOrgRole(ctx context.Context, orgID, userID, orgRoleID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM org_user_roles WHERE organization_id = ? AND user_id = ? AND org_role_id = ?`,
		orgID, userID, orgRoleID,
	)
	return err
}

func (s *SQLiteStore) GetOrgUserRoles(ctx context.Context, userID, orgID string) ([]*OrgRole, error) {
	return s.GetOrgRolesByUserID(ctx, userID, orgID)
}

// scanOrgRole scans a single org_roles row from a *sql.Rows cursor.
func scanOrgRole(rows interface {
	Scan(dest ...any) error
}) (*OrgRole, error) {
	var r OrgRole
	var isBuiltin int
	var description *string
	var createdAt, updatedAt string
	if err := rows.Scan(&r.ID, &r.OrganizationID, &r.Name, &description, &isBuiltin, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	if description != nil {
		r.Description = *description
	}
	r.IsBuiltin = isBuiltin != 0
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &r, nil
}
