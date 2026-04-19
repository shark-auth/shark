package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// --- Authorization Codes ---

func (s *SQLiteStore) CreateAuthorizationCode(ctx context.Context, code *OAuthAuthorizationCode) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_authorization_codes (code_hash, client_id, user_id, redirect_uri,
			scope, code_challenge, code_challenge_method, resource, authorization_details,
			nonce, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		code.CodeHash, code.ClientID, code.UserID, code.RedirectURI,
		code.Scope, code.CodeChallenge, code.CodeChallengeMethod,
		code.Resource, code.AuthorizationDetails, code.Nonce,
		code.ExpiresAt.UTC().Format(time.RFC3339), code.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetAuthorizationCode(ctx context.Context, codeHash string) (*OAuthAuthorizationCode, error) {
	var c OAuthAuthorizationCode
	var expiresAt, createdAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT code_hash, client_id, user_id, redirect_uri, scope, code_challenge,
			code_challenge_method, resource, authorization_details, nonce, expires_at, created_at
		FROM oauth_authorization_codes WHERE code_hash = ?`, codeHash).Scan(
		&c.CodeHash, &c.ClientID, &c.UserID, &c.RedirectURI, &c.Scope,
		&c.CodeChallenge, &c.CodeChallengeMethod, &c.Resource, &c.AuthorizationDetails,
		&c.Nonce, &expiresAt, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("authorization code not found")
	}
	if err != nil {
		return nil, err
	}
	c.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &c, nil
}

func (s *SQLiteStore) DeleteAuthorizationCode(ctx context.Context, codeHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oauth_authorization_codes WHERE code_hash = ?`, codeHash)
	return err
}

func (s *SQLiteStore) DeleteExpiredAuthorizationCodes(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM oauth_authorization_codes WHERE expires_at < ?`,
		time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// --- OAuth Tokens ---

func (s *SQLiteStore) CreateOAuthToken(ctx context.Context, token *OAuthToken) error {
	// agent_id and user_id are FK columns; pass NULL when unset so the FK
	// constraint is not violated by an empty string (which is never a valid ID).
	var agentID, userID interface{}
	if token.AgentID != "" {
		agentID = token.AgentID
	}
	if token.UserID != "" {
		userID = token.UserID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_tokens (id, jti, client_id, agent_id, user_id, token_type,
			token_hash, scope, audience, authorization_details, dpop_jkt,
			delegation_subject, delegation_actor, family_id, expires_at, created_at, revoked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.JTI, token.ClientID, agentID, userID,
		token.TokenType, token.TokenHash, token.Scope, token.Audience,
		token.AuthorizationDetails, token.DPoPJKT, token.DelegationSubject,
		token.DelegationActor, token.FamilyID,
		token.ExpiresAt.UTC().Format(time.RFC3339), token.CreatedAt.UTC().Format(time.RFC3339),
		nil, // revoked_at
	)
	return err
}

func (s *SQLiteStore) GetOAuthTokenByJTI(ctx context.Context, jti string) (*OAuthToken, error) {
	return s.scanOAuthToken(s.db.QueryRowContext(ctx, `
		SELECT id, jti, client_id, agent_id, user_id, token_type, token_hash,
			scope, audience, authorization_details, dpop_jkt, delegation_subject,
			delegation_actor, family_id, expires_at, created_at, revoked_at
		FROM oauth_tokens WHERE jti = ?`, jti))
}

func (s *SQLiteStore) GetOAuthTokenByHash(ctx context.Context, tokenHash string) (*OAuthToken, error) {
	return s.scanOAuthToken(s.db.QueryRowContext(ctx, `
		SELECT id, jti, client_id, agent_id, user_id, token_type, token_hash,
			scope, audience, authorization_details, dpop_jkt, delegation_subject,
			delegation_actor, family_id, expires_at, created_at, revoked_at
		FROM oauth_tokens WHERE token_hash = ?`, tokenHash))
}

func (s *SQLiteStore) RevokeOAuthToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *SQLiteStore) RevokeOAuthTokensByClientID(ctx context.Context, clientID string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE client_id = ? AND revoked_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), clientID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) RevokeOAuthTokenFamily(ctx context.Context, familyID string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET revoked_at = ? WHERE family_id = ? AND revoked_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), familyID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) ListOAuthTokensByAgentID(ctx context.Context, agentID string, limit int) ([]*OAuthToken, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, jti, client_id, agent_id, user_id, token_type, token_hash,
			scope, audience, authorization_details, dpop_jkt, delegation_subject,
			delegation_actor, family_id, expires_at, created_at, revoked_at
		FROM oauth_tokens WHERE agent_id = ? ORDER BY created_at DESC LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*OAuthToken
	for rows.Next() {
		t, err := s.scanOAuthTokenFromRows(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *SQLiteStore) DeleteExpiredOAuthTokens(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE expires_at < ? AND revoked_at IS NOT NULL`,
		time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) UpdateOAuthTokenDPoPJKT(ctx context.Context, id string, jkt string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_tokens SET dpop_jkt = ? WHERE id = ?`, jkt, id)
	return err
}

func (s *SQLiteStore) scanOAuthToken(row *sql.Row) (*OAuthToken, error) {
	var t OAuthToken
	var expiresAt, createdAt string
	var revokedAt *string
	var agentID, userID sql.NullString
	err := row.Scan(
		&t.ID, &t.JTI, &t.ClientID, &agentID, &userID, &t.TokenType,
		&t.TokenHash, &t.Scope, &t.Audience, &t.AuthorizationDetails,
		&t.DPoPJKT, &t.DelegationSubject, &t.DelegationActor, &t.FamilyID,
		&expiresAt, &createdAt, &revokedAt,
	)
	if err != nil {
		return nil, err
	}
	if agentID.Valid {
		t.AgentID = agentID.String
	}
	if userID.Valid {
		t.UserID = userID.String
	}
	t.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if revokedAt != nil {
		ra, _ := time.Parse(time.RFC3339, *revokedAt)
		t.RevokedAt = &ra
	}
	return &t, nil
}

func (s *SQLiteStore) scanOAuthTokenFromRows(rows *sql.Rows) (*OAuthToken, error) {
	var t OAuthToken
	var expiresAt, createdAt string
	var revokedAt *string
	var agentID, userID sql.NullString
	err := rows.Scan(
		&t.ID, &t.JTI, &t.ClientID, &agentID, &userID, &t.TokenType,
		&t.TokenHash, &t.Scope, &t.Audience, &t.AuthorizationDetails,
		&t.DPoPJKT, &t.DelegationSubject, &t.DelegationActor, &t.FamilyID,
		&expiresAt, &createdAt, &revokedAt,
	)
	if err != nil {
		return nil, err
	}
	if agentID.Valid {
		t.AgentID = agentID.String
	}
	if userID.Valid {
		t.UserID = userID.String
	}
	t.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if revokedAt != nil {
		ra, _ := time.Parse(time.RFC3339, *revokedAt)
		t.RevokedAt = &ra
	}
	return &t, nil
}

// --- Consents ---

func (s *SQLiteStore) CreateOAuthConsent(ctx context.Context, consent *OAuthConsent) error {
	var expiresAt *string
	if consent.ExpiresAt != nil {
		s := consent.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &s
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_consents (id, user_id, client_id, scope, authorization_details,
			granted_at, expires_at, revoked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		consent.ID, consent.UserID, consent.ClientID, consent.Scope,
		consent.AuthorizationDetails, consent.GrantedAt.UTC().Format(time.RFC3339),
		expiresAt, nil,
	)
	return err
}

func (s *SQLiteStore) GetActiveConsent(ctx context.Context, userID, clientID string) (*OAuthConsent, error) {
	var c OAuthConsent
	var grantedAt string
	var expiresAt, revokedAt *string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, client_id, scope, authorization_details, granted_at, expires_at, revoked_at
		FROM oauth_consents WHERE user_id = ? AND client_id = ? AND revoked_at IS NULL`,
		userID, clientID).Scan(
		&c.ID, &c.UserID, &c.ClientID, &c.Scope, &c.AuthorizationDetails,
		&grantedAt, &expiresAt, &revokedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // no active consent is not an error
	}
	if err != nil {
		return nil, err
	}
	c.GrantedAt, _ = time.Parse(time.RFC3339, grantedAt)
	if expiresAt != nil {
		ea, _ := time.Parse(time.RFC3339, *expiresAt)
		c.ExpiresAt = &ea
	}
	return &c, nil
}

func (s *SQLiteStore) ListConsentsByUserID(ctx context.Context, userID string) ([]*OAuthConsent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, client_id, scope, authorization_details, granted_at, expires_at, revoked_at
		FROM oauth_consents WHERE user_id = ? AND revoked_at IS NULL ORDER BY granted_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var consents []*OAuthConsent
	for rows.Next() {
		var c OAuthConsent
		var grantedAt string
		var expiresAt, revokedAt *string
		if err := rows.Scan(&c.ID, &c.UserID, &c.ClientID, &c.Scope,
			&c.AuthorizationDetails, &grantedAt, &expiresAt, &revokedAt); err != nil {
			return nil, err
		}
		c.GrantedAt, _ = time.Parse(time.RFC3339, grantedAt)
		if expiresAt != nil {
			ea, _ := time.Parse(time.RFC3339, *expiresAt)
			c.ExpiresAt = &ea
		}
		consents = append(consents, &c)
	}
	return consents, rows.Err()
}

func (s *SQLiteStore) RevokeOAuthConsent(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_consents SET revoked_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// --- Device Codes ---

func (s *SQLiteStore) CreateDeviceCode(ctx context.Context, dc *OAuthDeviceCode) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_device_codes (device_code_hash, user_code, client_id, scope,
			resource, status, poll_interval, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		dc.DeviceCodeHash, dc.UserCode, dc.ClientID, dc.Scope, dc.Resource,
		dc.Status, dc.PollInterval,
		dc.ExpiresAt.UTC().Format(time.RFC3339), dc.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*OAuthDeviceCode, error) {
	return s.scanDeviceCode(s.db.QueryRowContext(ctx, `
		SELECT device_code_hash, user_code, client_id, scope, resource, user_id,
			status, last_polled_at, poll_interval, expires_at, created_at
		FROM oauth_device_codes WHERE user_code = ?`, userCode))
}

func (s *SQLiteStore) GetDeviceCodeByHash(ctx context.Context, hash string) (*OAuthDeviceCode, error) {
	return s.scanDeviceCode(s.db.QueryRowContext(ctx, `
		SELECT device_code_hash, user_code, client_id, scope, resource, user_id,
			status, last_polled_at, poll_interval, expires_at, created_at
		FROM oauth_device_codes WHERE device_code_hash = ?`, hash))
}

func (s *SQLiteStore) UpdateDeviceCodeStatus(ctx context.Context, hash string, status string, userID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_device_codes SET status = ?, user_id = ? WHERE device_code_hash = ?`,
		status, userID, hash)
	return err
}

func (s *SQLiteStore) UpdateDeviceCodePolledAt(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oauth_device_codes SET last_polled_at = ? WHERE device_code_hash = ?`,
		time.Now().UTC().Format(time.RFC3339), hash)
	return err
}

func (s *SQLiteStore) DeleteExpiredDeviceCodes(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM oauth_device_codes WHERE expires_at < ?`,
		time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) scanDeviceCode(row *sql.Row) (*OAuthDeviceCode, error) {
	var dc OAuthDeviceCode
	var expiresAt, createdAt string
	var lastPolledAt *string
	var userID *string
	err := row.Scan(
		&dc.DeviceCodeHash, &dc.UserCode, &dc.ClientID, &dc.Scope, &dc.Resource,
		&userID, &dc.Status, &lastPolledAt, &dc.PollInterval, &expiresAt, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device code not found")
	}
	if err != nil {
		return nil, err
	}
	dc.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	dc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if lastPolledAt != nil {
		lp, _ := time.Parse(time.RFC3339, *lastPolledAt)
		dc.LastPolledAt = &lp
	}
	if userID != nil {
		dc.UserID = *userID
	}
	return &dc, nil
}

// --- DCR Clients ---

func (s *SQLiteStore) CreateDCRClient(ctx context.Context, client *OAuthDCRClient) error {
	var expiresAt *string
	if client.ExpiresAt != nil {
		s := client.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &s
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_dcr_clients (client_id, registration_token_hash, client_metadata, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		client.ClientID, client.RegistrationTokenHash, client.ClientMetadata,
		client.CreatedAt.UTC().Format(time.RFC3339), expiresAt,
	)
	return err
}

func (s *SQLiteStore) GetDCRClient(ctx context.Context, clientID string) (*OAuthDCRClient, error) {
	var c OAuthDCRClient
	var createdAt string
	var expiresAt *string
	err := s.db.QueryRowContext(ctx, `
		SELECT client_id, registration_token_hash, client_metadata, created_at, expires_at
		FROM oauth_dcr_clients WHERE client_id = ?`, clientID).Scan(
		&c.ClientID, &c.RegistrationTokenHash, &c.ClientMetadata, &createdAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("DCR client not found")
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if expiresAt != nil {
		ea, _ := time.Parse(time.RFC3339, *expiresAt)
		c.ExpiresAt = &ea
	}
	return &c, nil
}

func (s *SQLiteStore) UpdateDCRClient(ctx context.Context, client *OAuthDCRClient) error {
	var expiresAt *string
	if client.ExpiresAt != nil {
		s := client.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAt = &s
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE oauth_dcr_clients SET client_metadata = ?, expires_at = ? WHERE client_id = ?`,
		client.ClientMetadata, expiresAt, client.ClientID,
	)
	return err
}

func (s *SQLiteStore) DeleteDCRClient(ctx context.Context, clientID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oauth_dcr_clients WHERE client_id = ?`, clientID)
	return err
}
