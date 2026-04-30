package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// CreateAgent inserts a new agent into the database.
func (s *SQLiteStore) CreateAgent(ctx context.Context, agent *Agent) error {
	redirectURIs, _ := json.Marshal(agent.RedirectURIs)
	grantTypes, _ := json.Marshal(agent.GrantTypes)
	responseTypes, _ := json.Marshal(agent.ResponseTypes)
	scopes, _ := json.Marshal(agent.Scopes)
	metadata, _ := json.Marshal(agent.Metadata)

	// created_by is a FK to users(id) — pass NULL when empty so INSERT doesn't
	// violate the foreign-key constraint (empty string is not a valid user ID).
	var createdBy interface{}
	if agent.CreatedBy != "" {
		createdBy = agent.CreatedBy
	}

	_, err := s.writer.ExecContext(ctx, `
		INSERT INTO agents (id, name, description, client_id, client_secret_hash,
			client_type, auth_method, jwks, jwks_uri, redirect_uris, grant_types,
			response_types, scopes, token_lifetime, metadata, logo_uri, homepage_uri,
			active, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.ID, agent.Name, agent.Description, agent.ClientID, agent.ClientSecretHash,
		agent.ClientType, agent.AuthMethod, agent.JWKS, agent.JWKSURI,
		string(redirectURIs), string(grantTypes), string(responseTypes), string(scopes),
		agent.TokenLifetime, string(metadata), agent.LogoURI, agent.HomepageURI,
		agent.Active, createdBy,
		agent.CreatedAt.UTC().Format(time.RFC3339), agent.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetAgentByID retrieves an agent by its internal ID.
func (s *SQLiteStore) GetAgentByID(ctx context.Context, id string) (*Agent, error) {
	return s.scanAgent(s.reader.QueryRowContext(ctx, `SELECT * FROM agents WHERE id = ?`, id))
}

// GetAgentByClientID retrieves an agent by its OAuth client_id.
func (s *SQLiteStore) GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error) {
	return s.scanAgent(s.reader.QueryRowContext(ctx, `SELECT * FROM agents WHERE client_id = ?`, clientID))
}

// ListAgents returns agents with optional filtering and pagination.
func (s *SQLiteStore) ListAgents(ctx context.Context, opts ListAgentsOpts) ([]*Agent, int, error) {
	// When filtering by AuthorizedByUser we add an IN-subquery against
	// oauth_consents to the WHERE clause directly. The previous attempt
	// embedded a WHERE inside the FROM clause and produced invalid SQL.
	fromClause := "agents"
	where := "1=1"
	args := []interface{}{}

	if opts.AuthorizedByUser != nil {
		where += " AND client_id IN (SELECT DISTINCT client_id FROM oauth_consents WHERE user_id = ? AND revoked_at IS NULL)"
		args = append(args, *opts.AuthorizedByUser)
	}

	if opts.Search != "" {
		where += " AND (name LIKE ? OR description LIKE ?)"
		pattern := "%" + opts.Search + "%"
		args = append(args, pattern, pattern)
	}
	if opts.Active != nil {
		where += " AND active = ?"
		if *opts.Active {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if opts.CreatedByUserID != nil {
		where += " AND created_by = ?"
		args = append(args, *opts.CreatedByUserID)
	}

	countQuery := "SELECT COUNT(*) FROM " + fromClause + " WHERE " + where
	selectQuery := "SELECT * FROM " + fromClause + " WHERE " + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	// Count
	var total int
	err := s.reader.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Query
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	queryArgs := append(args, limit, opts.Offset)
	rows, err := s.reader.QueryContext(ctx, selectQuery, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		a, err := s.scanAgentFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		agents = append(agents, a)
	}
	return agents, total, rows.Err()
}

// ListAgentsByUserID returns all agents created by the given user.
func (s *SQLiteStore) ListAgentsByUserID(ctx context.Context, userID string) ([]*Agent, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT * FROM agents WHERE created_by = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		a, err := s.scanAgentFromRows(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

// UpdateAgent updates an existing agent.
func (s *SQLiteStore) UpdateAgent(ctx context.Context, agent *Agent) error {
	redirectURIs, _ := json.Marshal(agent.RedirectURIs)
	grantTypes, _ := json.Marshal(agent.GrantTypes)
	responseTypes, _ := json.Marshal(agent.ResponseTypes)
	scopes, _ := json.Marshal(agent.Scopes)
	metadata, _ := json.Marshal(agent.Metadata)

	_, err := s.writer.ExecContext(ctx, `
		UPDATE agents SET name=?, description=?, client_secret_hash=?,
			client_type=?, auth_method=?, jwks=?, jwks_uri=?, redirect_uris=?,
			grant_types=?, response_types=?, scopes=?, token_lifetime=?,
			metadata=?, logo_uri=?, homepage_uri=?, active=?,
			updated_at=?
		WHERE id=?`,
		agent.Name, agent.Description, agent.ClientSecretHash,
		agent.ClientType, agent.AuthMethod, agent.JWKS, agent.JWKSURI,
		string(redirectURIs), string(grantTypes), string(responseTypes), string(scopes),
		agent.TokenLifetime, string(metadata), agent.LogoURI, agent.HomepageURI,
		agent.Active, time.Now().UTC().Format(time.RFC3339),
		agent.ID,
	)
	return err
}

// UpdateAgentSecret replaces the client_secret_hash for an agent.
func (s *SQLiteStore) UpdateAgentSecret(ctx context.Context, id, secretHash string) error {
	_, err := s.writer.ExecContext(ctx,
		`UPDATE agents SET client_secret_hash=?, updated_at=? WHERE id=?`,
		secretHash, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

// DeactivateAgent sets active=0 for an agent.
func (s *SQLiteStore) DeactivateAgent(ctx context.Context, id string) error {
	_, err := s.writer.ExecContext(ctx, `UPDATE agents SET active=0, updated_at=? WHERE id=?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// RotateDCRClientSecret updates the client_secret_hash for a DCR-registered agent,
// preserving the previous hash as old_secret_hash valid until expiresAt.
// Uses a targeted UPDATE so no SELECT * re-scan is needed.
func (s *SQLiteStore) RotateDCRClientSecret(ctx context.Context, agentID, newSecretHash, oldSecretHash string, oldSecretExpiresAt time.Time) error {
	expiresAtStr := oldSecretExpiresAt.UTC().Format(time.RFC3339)
	_, err := s.writer.ExecContext(ctx, `
		UPDATE agents SET
			client_secret_hash    = ?,
			old_secret_hash       = ?,
			old_secret_expires_at = ?,
			updated_at            = ?
		WHERE id = ?`,
		newSecretHash, oldSecretHash, expiresAtStr,
		time.Now().UTC().Format(time.RFC3339),
		agentID,
	)
	return err
}

// scanAgent scans a single agent row.
func (s *SQLiteStore) scanAgent(row *sql.Row) (*Agent, error) {
	var a Agent
	var redirectURIs, grantTypes, responseTypes, scopes, metadata string
	var createdAt, updatedAt string
	var active int
	var jwks, jwksURI, logoURI, homepageURI sql.NullString
	var createdBy sql.NullString
	var oldSecretHash sql.NullString
	var oldSecretExpiresAt sql.NullString

	err := row.Scan(
		&a.ID, &a.Name, &a.Description, &a.ClientID, &a.ClientSecretHash,
		&a.ClientType, &a.AuthMethod, &jwks, &jwksURI,
		&redirectURIs, &grantTypes, &responseTypes, &scopes,
		&a.TokenLifetime, &metadata, &logoURI, &homepageURI,
		&active, &createdBy,
		&createdAt, &updatedAt,
		&oldSecretHash, &oldSecretExpiresAt,
	)
	if err != nil {
		// Propagate sql.ErrNoRows unwrapped so callers can `errors.Is` it.
		return nil, err
	}

	if jwks.Valid {
		a.JWKS = jwks.String
	}
	if jwksURI.Valid {
		a.JWKSURI = jwksURI.String
	}
	if logoURI.Valid {
		a.LogoURI = logoURI.String
	}
	if homepageURI.Valid {
		a.HomepageURI = homepageURI.String
	}
	if createdBy.Valid {
		a.CreatedBy = createdBy.String
	}
	if oldSecretHash.Valid {
		a.OldSecretHash = oldSecretHash.String
	}
	if oldSecretExpiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, oldSecretExpiresAt.String)
		if t.IsZero() {
			// Try SQLite CURRENT_TIMESTAMP format
			t, _ = time.Parse("2006-01-02 15:04:05", oldSecretExpiresAt.String)
		}
		if !t.IsZero() {
			a.OldSecretExpiresAt = &t
		}
	}
	a.Active = active == 1
	json.Unmarshal([]byte(redirectURIs), &a.RedirectURIs)   //#nosec G104
	json.Unmarshal([]byte(grantTypes), &a.GrantTypes)       //#nosec G104
	json.Unmarshal([]byte(responseTypes), &a.ResponseTypes) //#nosec G104
	json.Unmarshal([]byte(scopes), &a.Scopes)               //#nosec G104
	json.Unmarshal([]byte(metadata), &a.Metadata)           //#nosec G104
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if a.RedirectURIs == nil {
		a.RedirectURIs = []string{}
	}
	if a.GrantTypes == nil {
		a.GrantTypes = []string{}
	}
	if a.ResponseTypes == nil {
		a.ResponseTypes = []string{}
	}
	if a.Scopes == nil {
		a.Scopes = []string{}
	}
	if a.Metadata == nil {
		a.Metadata = map[string]any{}
	}

	return &a, nil
}

// scanAgentFromRows scans an agent from sql.Rows.
func (s *SQLiteStore) scanAgentFromRows(rows *sql.Rows) (*Agent, error) {
	var a Agent
	var redirectURIs, grantTypes, responseTypes, scopes, metadata string
	var createdAt, updatedAt string
	var active int
	var jwks, jwksURI, logoURI, homepageURI sql.NullString
	var createdBy sql.NullString
	var oldSecretHash sql.NullString
	var oldSecretExpiresAt sql.NullString

	err := rows.Scan(
		&a.ID, &a.Name, &a.Description, &a.ClientID, &a.ClientSecretHash,
		&a.ClientType, &a.AuthMethod, &jwks, &jwksURI,
		&redirectURIs, &grantTypes, &responseTypes, &scopes,
		&a.TokenLifetime, &metadata, &logoURI, &homepageURI,
		&active, &createdBy,
		&createdAt, &updatedAt,
		&oldSecretHash, &oldSecretExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	if jwks.Valid {
		a.JWKS = jwks.String
	}
	if jwksURI.Valid {
		a.JWKSURI = jwksURI.String
	}
	if logoURI.Valid {
		a.LogoURI = logoURI.String
	}
	if homepageURI.Valid {
		a.HomepageURI = homepageURI.String
	}
	if createdBy.Valid {
		a.CreatedBy = createdBy.String
	}
	if oldSecretHash.Valid {
		a.OldSecretHash = oldSecretHash.String
	}
	if oldSecretExpiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, oldSecretExpiresAt.String)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02 15:04:05", oldSecretExpiresAt.String)
		}
		if !t.IsZero() {
			a.OldSecretExpiresAt = &t
		}
	}
	a.Active = active == 1
	json.Unmarshal([]byte(redirectURIs), &a.RedirectURIs)   //#nosec G104
	json.Unmarshal([]byte(grantTypes), &a.GrantTypes)       //#nosec G104
	json.Unmarshal([]byte(responseTypes), &a.ResponseTypes) //#nosec G104
	json.Unmarshal([]byte(scopes), &a.Scopes)               //#nosec G104
	json.Unmarshal([]byte(metadata), &a.Metadata)           //#nosec G104
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if a.RedirectURIs == nil {
		a.RedirectURIs = []string{}
	}
	if a.GrantTypes == nil {
		a.GrantTypes = []string{}
	}
	if a.ResponseTypes == nil {
		a.ResponseTypes = []string{}
	}
	if a.Scopes == nil {
		a.Scopes = []string{}
	}
	if a.Metadata == nil {
		a.Metadata = map[string]any{}
	}

	return &a, nil
}
