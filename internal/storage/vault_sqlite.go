package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// --- Vault Providers ---

func (s *SQLiteStore) CreateVaultProvider(ctx context.Context, p *VaultProvider) error {
	scopesJSON, err := marshalStringSlice(p.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}
	var iconURL interface{}
	if p.IconURL != "" {
		iconURL = p.IconURL
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO vault_providers
		 (id, name, display_name, auth_url, token_url, client_id,
		  client_secret_enc, scopes, icon_url, active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.DisplayName, p.AuthURL, p.TokenURL, p.ClientID,
		p.ClientSecretEnc, scopesJSON, iconURL, boolToInt(p.Active),
		p.CreatedAt.UTC().Format(time.RFC3339),
		p.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetVaultProviderByID(ctx context.Context, id string) (*VaultProvider, error) {
	return s.scanVaultProvider(s.db.QueryRowContext(ctx,
		`SELECT id, name, display_name, auth_url, token_url, client_id,
		        client_secret_enc, scopes, icon_url, active, created_at, updated_at
		 FROM vault_providers WHERE id = ?`, id))
}

func (s *SQLiteStore) GetVaultProviderByName(ctx context.Context, name string) (*VaultProvider, error) {
	return s.scanVaultProvider(s.db.QueryRowContext(ctx,
		`SELECT id, name, display_name, auth_url, token_url, client_id,
		        client_secret_enc, scopes, icon_url, active, created_at, updated_at
		 FROM vault_providers WHERE name = ?`, name))
}

func (s *SQLiteStore) ListVaultProviders(ctx context.Context, activeOnly bool) ([]*VaultProvider, error) {
	query := `SELECT id, name, display_name, auth_url, token_url, client_id,
	                 client_secret_enc, scopes, icon_url, active, created_at, updated_at
	          FROM vault_providers`
	if activeOnly {
		query += ` WHERE active = 1`
	}
	query += ` ORDER BY display_name ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*VaultProvider
	for rows.Next() {
		p, err := s.scanVaultProviderFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateVaultProvider(ctx context.Context, p *VaultProvider) error {
	scopesJSON, err := marshalStringSlice(p.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}
	var iconURL interface{}
	if p.IconURL != "" {
		iconURL = p.IconURL
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE vault_providers SET
		   name = ?, display_name = ?, auth_url = ?, token_url = ?,
		   client_id = ?, client_secret_enc = ?, scopes = ?, icon_url = ?,
		   active = ?, updated_at = ?
		 WHERE id = ?`,
		p.Name, p.DisplayName, p.AuthURL, p.TokenURL,
		p.ClientID, p.ClientSecretEnc, scopesJSON, iconURL,
		boolToInt(p.Active), time.Now().UTC().Format(time.RFC3339),
		p.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteVaultProvider(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM vault_providers WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) scanVaultProvider(row *sql.Row) (*VaultProvider, error) {
	var p VaultProvider
	var scopesJSON string
	var iconURL sql.NullString
	var active int
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.AuthURL, &p.TokenURL, &p.ClientID,
		&p.ClientSecretEnc, &scopesJSON, &iconURL, &active,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return unmarshalVaultProvider(&p, scopesJSON, iconURL, active, createdAtStr, updatedAtStr)
}

func (s *SQLiteStore) scanVaultProviderFromRows(rows *sql.Rows) (*VaultProvider, error) {
	var p VaultProvider
	var scopesJSON string
	var iconURL sql.NullString
	var active int
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.AuthURL, &p.TokenURL, &p.ClientID,
		&p.ClientSecretEnc, &scopesJSON, &iconURL, &active,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return unmarshalVaultProvider(&p, scopesJSON, iconURL, active, createdAtStr, updatedAtStr)
}

func unmarshalVaultProvider(
	p *VaultProvider,
	scopesJSON string,
	iconURL sql.NullString,
	active int,
	createdAtStr, updatedAtStr string,
) (*VaultProvider, error) {
	p.Active = active != 0
	if iconURL.Valid {
		p.IconURL = iconURL.String
	}

	if err := json.Unmarshal([]byte(scopesJSON), &p.Scopes); err != nil {
		return nil, fmt.Errorf("unmarshal scopes: %w", err)
	}
	if p.Scopes == nil {
		p.Scopes = []string{}
	}

	var err error
	p.CreatedAt, err = parseVaultTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at %q: %w", createdAtStr, err)
	}
	p.UpdatedAt, err = parseVaultTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at %q: %w", updatedAtStr, err)
	}
	return p, nil
}

// --- Vault Connections ---

func (s *SQLiteStore) CreateVaultConnection(ctx context.Context, c *VaultConnection) error {
	scopesJSON, err := marshalStringSlice(c.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}
	metaJSON, err := marshalMetadata(c.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	var refreshTok interface{}
	if c.RefreshTokenEnc != "" {
		refreshTok = c.RefreshTokenEnc
	}
	var expiresAt interface{}
	if c.ExpiresAt != nil {
		expiresAt = c.ExpiresAt.UTC().Format(time.RFC3339)
	}
	var lastRefreshed interface{}
	if c.LastRefreshedAt != nil {
		lastRefreshed = c.LastRefreshedAt.UTC().Format(time.RFC3339)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO vault_connections
		 (id, provider_id, user_id, access_token_enc, refresh_token_enc,
		  token_type, scopes, expires_at, metadata, needs_reauth,
		  last_refreshed_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.ProviderID, c.UserID, c.AccessTokenEnc, refreshTok,
		c.TokenType, scopesJSON, expiresAt, metaJSON, boolToInt(c.NeedsReauth),
		lastRefreshed,
		c.CreatedAt.UTC().Format(time.RFC3339),
		c.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetVaultConnectionByID(ctx context.Context, id string) (*VaultConnection, error) {
	return s.scanVaultConnection(s.db.QueryRowContext(ctx,
		`SELECT id, provider_id, user_id, access_token_enc, refresh_token_enc,
		        token_type, scopes, expires_at, metadata, needs_reauth,
		        last_refreshed_at, created_at, updated_at
		 FROM vault_connections WHERE id = ?`, id))
}

func (s *SQLiteStore) GetVaultConnection(ctx context.Context, providerID, userID string) (*VaultConnection, error) {
	return s.scanVaultConnection(s.db.QueryRowContext(ctx,
		`SELECT id, provider_id, user_id, access_token_enc, refresh_token_enc,
		        token_type, scopes, expires_at, metadata, needs_reauth,
		        last_refreshed_at, created_at, updated_at
		 FROM vault_connections WHERE provider_id = ? AND user_id = ?`,
		providerID, userID))
}

func (s *SQLiteStore) ListVaultConnectionsByUserID(ctx context.Context, userID string) ([]*VaultConnection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, provider_id, user_id, access_token_enc, refresh_token_enc,
		        token_type, scopes, expires_at, metadata, needs_reauth,
		        last_refreshed_at, created_at, updated_at
		 FROM vault_connections WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*VaultConnection
	for rows.Next() {
		c, err := s.scanVaultConnectionFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListAllVaultConnections returns every vault connection across all users.
// Admin-scope view used by the dashboard connections tab. Encrypted token
// columns are still loaded but never serialized (json:"-" on the struct).
func (s *SQLiteStore) ListAllVaultConnections(ctx context.Context) ([]*VaultConnection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, provider_id, user_id, access_token_enc, refresh_token_enc,
		        token_type, scopes, expires_at, metadata, needs_reauth,
		        last_refreshed_at, created_at, updated_at
		 FROM vault_connections ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*VaultConnection
	for rows.Next() {
		c, err := s.scanVaultConnectionFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListVaultConnectionsByProviderID(ctx context.Context, providerID string) ([]*VaultConnection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, provider_id, user_id, access_token_enc, refresh_token_enc,
		        token_type, scopes, expires_at, metadata, needs_reauth,
		        last_refreshed_at, created_at, updated_at
		 FROM vault_connections WHERE provider_id = ? ORDER BY created_at DESC`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*VaultConnection
	for rows.Next() {
		c, err := s.scanVaultConnectionFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateVaultConnection(ctx context.Context, c *VaultConnection) error {
	scopesJSON, err := marshalStringSlice(c.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}
	metaJSON, err := marshalMetadata(c.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	var refreshTok interface{}
	if c.RefreshTokenEnc != "" {
		refreshTok = c.RefreshTokenEnc
	}
	var expiresAt interface{}
	if c.ExpiresAt != nil {
		expiresAt = c.ExpiresAt.UTC().Format(time.RFC3339)
	}
	var lastRefreshed interface{}
	if c.LastRefreshedAt != nil {
		lastRefreshed = c.LastRefreshedAt.UTC().Format(time.RFC3339)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE vault_connections SET
		   access_token_enc = ?, refresh_token_enc = ?,
		   token_type = ?, scopes = ?, expires_at = ?,
		   metadata = ?, needs_reauth = ?, last_refreshed_at = ?,
		   updated_at = ?
		 WHERE id = ?`,
		c.AccessTokenEnc, refreshTok,
		c.TokenType, scopesJSON, expiresAt,
		metaJSON, boolToInt(c.NeedsReauth), lastRefreshed,
		time.Now().UTC().Format(time.RFC3339),
		c.ID,
	)
	return err
}

// UpdateVaultConnectionTokens rotates the encrypted access + refresh tokens in
// a single statement and clears the needs_reauth flag (a successful refresh
// proves the user doesn't need to re-consent). last_refreshed_at stamps now.
func (s *SQLiteStore) UpdateVaultConnectionTokens(
	ctx context.Context,
	id, accessEnc, refreshEnc string,
	expiresAt *time.Time,
) error {
	var refreshTok interface{}
	if refreshEnc != "" {
		refreshTok = refreshEnc
	}
	var expiresAtArg interface{}
	if expiresAt != nil {
		expiresAtArg = expiresAt.UTC().Format(time.RFC3339)
	}
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.ExecContext(ctx,
		`UPDATE vault_connections SET
		   access_token_enc = ?, refresh_token_enc = ?, expires_at = ?,
		   last_refreshed_at = ?, needs_reauth = 0, updated_at = ?
		 WHERE id = ?`,
		accessEnc, refreshTok, expiresAtArg, now, now, id,
	)
	return err
}

// MarkVaultConnectionNeedsReauth toggles the needs_reauth flag. Called from
// the refresh pipeline when the provider rejects the refresh_token, so the UI
// can nudge the user to re-connect.
func (s *SQLiteStore) MarkVaultConnectionNeedsReauth(ctx context.Context, id string, needs bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE vault_connections SET needs_reauth = ?, updated_at = ? WHERE id = ?`,
		boolToInt(needs), time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

func (s *SQLiteStore) DeleteVaultConnection(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM vault_connections WHERE id = ?`, id)
	return err
}

// ListAgentsByVaultRetrieval returns agents that have ever fetched a token from
// the given vault connection. It queries the audit_logs table for
// vault.token.retrieved events whose target_id matches connectionID, then
// resolves each distinct actor_id to an Agent row.
func (s *SQLiteStore) ListAgentsByVaultRetrieval(ctx context.Context, connectionID string) ([]*Agent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT actor_id FROM audit_logs WHERE action = 'vault.token.retrieved' AND target_id = ?`,
		connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actorIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if id != "" {
			actorIDs = append(actorIDs, id)
		}
	}

	agents := make([]*Agent, 0, len(actorIDs))
	for _, aid := range actorIDs {
		ag, err := s.GetAgentByID(ctx, aid)
		if err != nil || ag == nil {
			continue
		}
		agents = append(agents, ag)
	}
	return agents, nil
}

func (s *SQLiteStore) scanVaultConnection(row *sql.Row) (*VaultConnection, error) {
	var c VaultConnection
	var refreshTok sql.NullString
	var scopesJSON, metaJSON string
	var expiresAtStr, lastRefreshedStr sql.NullString
	var needsReauth int
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&c.ID, &c.ProviderID, &c.UserID, &c.AccessTokenEnc, &refreshTok,
		&c.TokenType, &scopesJSON, &expiresAtStr, &metaJSON, &needsReauth,
		&lastRefreshedStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return unmarshalVaultConnection(&c, refreshTok, scopesJSON, metaJSON,
		expiresAtStr, lastRefreshedStr, needsReauth, createdAtStr, updatedAtStr)
}

func (s *SQLiteStore) scanVaultConnectionFromRows(rows *sql.Rows) (*VaultConnection, error) {
	var c VaultConnection
	var refreshTok sql.NullString
	var scopesJSON, metaJSON string
	var expiresAtStr, lastRefreshedStr sql.NullString
	var needsReauth int
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&c.ID, &c.ProviderID, &c.UserID, &c.AccessTokenEnc, &refreshTok,
		&c.TokenType, &scopesJSON, &expiresAtStr, &metaJSON, &needsReauth,
		&lastRefreshedStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return unmarshalVaultConnection(&c, refreshTok, scopesJSON, metaJSON,
		expiresAtStr, lastRefreshedStr, needsReauth, createdAtStr, updatedAtStr)
}

func unmarshalVaultConnection(
	c *VaultConnection,
	refreshTok sql.NullString,
	scopesJSON, metaJSON string,
	expiresAtStr, lastRefreshedStr sql.NullString,
	needsReauth int,
	createdAtStr, updatedAtStr string,
) (*VaultConnection, error) {
	if refreshTok.Valid {
		c.RefreshTokenEnc = refreshTok.String
	}
	c.NeedsReauth = needsReauth != 0

	if err := json.Unmarshal([]byte(scopesJSON), &c.Scopes); err != nil {
		return nil, fmt.Errorf("unmarshal scopes: %w", err)
	}
	if err := json.Unmarshal([]byte(metaJSON), &c.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	if c.Scopes == nil {
		c.Scopes = []string{}
	}
	if c.Metadata == nil {
		c.Metadata = map[string]any{}
	}

	if expiresAtStr.Valid && expiresAtStr.String != "" {
		t, err := parseVaultTime(expiresAtStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse expires_at %q: %w", expiresAtStr.String, err)
		}
		c.ExpiresAt = &t
	}
	if lastRefreshedStr.Valid && lastRefreshedStr.String != "" {
		t, err := parseVaultTime(lastRefreshedStr.String)
		if err != nil {
			return nil, fmt.Errorf("parse last_refreshed_at %q: %w", lastRefreshedStr.String, err)
		}
		c.LastRefreshedAt = &t
	}

	var err error
	c.CreatedAt, err = parseVaultTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at %q: %w", createdAtStr, err)
	}
	c.UpdatedAt, err = parseVaultTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at %q: %w", updatedAtStr, err)
	}
	return c, nil
}

// parseVaultTime accepts both RFC3339 (how we write) and SQLite's default
// CURRENT_TIMESTAMP format ("2006-01-02 15:04:05", no T separator).
func parseVaultTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}
