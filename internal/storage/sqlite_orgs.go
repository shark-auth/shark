package storage

import (
	"context"
	"database/sql"
	"strings"
)

// --- Organizations ---

func (s *SQLiteStore) CreateOrganization(ctx context.Context, o *Organization) error {
	if o.Metadata == "" {
		o.Metadata = "{}"
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO organizations (id, name, slug, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		o.ID, o.Name, o.Slug, o.Metadata, o.CreatedAt, o.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetOrganizationByID(ctx context.Context, id string) (*Organization, error) {
	return s.scanOrg(s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, metadata, created_at, updated_at FROM organizations WHERE id = ?`, id))
}

func (s *SQLiteStore) GetOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.scanOrg(s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, metadata, created_at, updated_at FROM organizations WHERE slug = ?`, slug))
}

func (s *SQLiteStore) UpdateOrganization(ctx context.Context, o *Organization) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE organizations SET name = ?, slug = ?, metadata = ?, updated_at = ? WHERE id = ?`,
		o.Name, o.Slug, o.Metadata, o.UpdatedAt, o.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteOrganization(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM organizations WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) ListOrganizationsByUserID(ctx context.Context, userID string) ([]*Organization, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.id, o.name, o.slug, o.metadata, o.created_at, o.updated_at
		 FROM organizations o
		 JOIN organization_members m ON m.organization_id = o.id
		 WHERE m.user_id = ?
		 ORDER BY o.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.Metadata, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListAllOrganizations(ctx context.Context) ([]*Organization, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, slug, metadata, created_at, updated_at
		 FROM organizations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.Metadata, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) scanOrg(row *sql.Row) (*Organization, error) {
	var o Organization
	if err := row.Scan(&o.ID, &o.Name, &o.Slug, &o.Metadata, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	return &o, nil
}

// --- Organization members ---

func (s *SQLiteStore) CreateOrganizationMember(ctx context.Context, m *OrganizationMember) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO organization_members (organization_id, user_id, role, joined_at)
		 VALUES (?, ?, ?, ?)`,
		m.OrganizationID, m.UserID, m.Role, m.JoinedAt,
	)
	return err
}

func (s *SQLiteStore) GetOrganizationMember(ctx context.Context, orgID, userID string) (*OrganizationMember, error) {
	var m OrganizationMember
	err := s.db.QueryRowContext(ctx,
		`SELECT organization_id, user_id, role, joined_at
		 FROM organization_members WHERE organization_id = ? AND user_id = ?`,
		orgID, userID,
	).Scan(&m.OrganizationID, &m.UserID, &m.Role, &m.JoinedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *SQLiteStore) UpdateOrganizationMemberRole(ctx context.Context, orgID, userID, role string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE organization_members SET role = ? WHERE organization_id = ? AND user_id = ?`,
		role, orgID, userID,
	)
	return err
}

func (s *SQLiteStore) DeleteOrganizationMember(ctx context.Context, orgID, userID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM organization_members WHERE organization_id = ? AND user_id = ?`,
		orgID, userID,
	)
	return err
}

func (s *SQLiteStore) ListOrganizationMembers(ctx context.Context, orgID string) ([]*OrganizationMemberWithUser, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT m.organization_id, m.user_id, m.role, m.joined_at,
		        COALESCE(u.email, ''), COALESCE(u.name, '')
		 FROM organization_members m
		 LEFT JOIN users u ON u.id = m.user_id
		 WHERE m.organization_id = ?
		 ORDER BY m.joined_at ASC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*OrganizationMemberWithUser
	for rows.Next() {
		var mw OrganizationMemberWithUser
		if err := rows.Scan(&mw.OrganizationID, &mw.UserID, &mw.Role, &mw.JoinedAt,
			&mw.UserEmail, &mw.UserName); err != nil {
			return nil, err
		}
		out = append(out, &mw)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) CountOrganizationMembers(ctx context.Context, orgID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM organization_members WHERE organization_id = ?`, orgID,
	).Scan(&n)
	return n, err
}

// CountOrganizationsByRole returns how many orgs the user belongs to with the
// given role. Used to prevent removing the last owner of an org.
func (s *SQLiteStore) CountOrganizationsByRole(ctx context.Context, userID, role string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM organization_members WHERE user_id = ? AND role = ?`,
		userID, role,
	).Scan(&n)
	return n, err
}

// --- Organization invitations ---

func (s *SQLiteStore) CreateOrganizationInvitation(ctx context.Context, inv *OrganizationInvitation) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO organization_invitations
		 (id, organization_id, email, role, token_hash, invited_by, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.OrganizationID, strings.ToLower(inv.Email), inv.Role, inv.TokenHash,
		inv.InvitedBy, inv.ExpiresAt, inv.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetOrganizationInvitationByID(ctx context.Context, id string) (*OrganizationInvitation, error) {
	var inv OrganizationInvitation
	err := s.db.QueryRowContext(ctx,
		`SELECT id, organization_id, email, role, token_hash, invited_by, accepted_at, expires_at, created_at
		 FROM organization_invitations WHERE id = ?`, id,
	).Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.TokenHash,
		&inv.InvitedBy, &inv.AcceptedAt, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// UpdateOrganizationInvitationToken rotates the stored token hash + expiry. Used
// when an admin resends an invitation so the previous link is invalidated and
// the new email carries a fresh, single-use token.
func (s *SQLiteStore) UpdateOrganizationInvitationToken(ctx context.Context, id, tokenHash, expiresAt string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE organization_invitations SET token_hash = ?, expires_at = ? WHERE id = ?`,
		tokenHash, expiresAt, id,
	)
	return err
}

// DeleteOrganizationInvitation removes a pending or accepted invitation row.
func (s *SQLiteStore) DeleteOrganizationInvitation(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM organization_invitations WHERE id = ?`, id,
	)
	return err
}

func (s *SQLiteStore) GetOrganizationInvitationByTokenHash(ctx context.Context, tokenHash string) (*OrganizationInvitation, error) {
	var inv OrganizationInvitation
	err := s.db.QueryRowContext(ctx,
		`SELECT id, organization_id, email, role, token_hash, invited_by, accepted_at, expires_at, created_at
		 FROM organization_invitations WHERE token_hash = ?`, tokenHash,
	).Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.TokenHash,
		&inv.InvitedBy, &inv.AcceptedAt, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (s *SQLiteStore) MarkOrganizationInvitationAccepted(ctx context.Context, id string, acceptedAt string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE organization_invitations SET accepted_at = ? WHERE id = ? AND accepted_at IS NULL`,
		acceptedAt, id,
	)
	return err
}

func (s *SQLiteStore) ListOrganizationInvitationsByOrgID(ctx context.Context, orgID string) ([]*OrganizationInvitation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, organization_id, email, role, token_hash, invited_by, accepted_at, expires_at, created_at
		 FROM organization_invitations WHERE organization_id = ?
		 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*OrganizationInvitation
	for rows.Next() {
		var inv OrganizationInvitation
		if err := rows.Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.TokenHash,
			&inv.InvitedBy, &inv.AcceptedAt, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &inv)
	}
	return out, rows.Err()
}
