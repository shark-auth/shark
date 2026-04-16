package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens a SQLite database at the given path and configures it
// with WAL mode and foreign keys enabled.
func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	// `:memory:` gives each sql connection its own private DB. Capping the pool
	// at 1 means everything the app does lands in the same in-memory instance.
	// Tests and concurrent goroutines (async email, webhook dispatcher) depend
	// on this. File-backed DBs stay on the default unlimited pool.
	if dsn == ":memory:" {
		db.SetMaxOpenConns(1)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	// Set pragmas via exec (modernc.org/sqlite doesn't support DSN pragmas)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying *sql.DB.
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// --- Users ---

func (s *SQLiteStore) CreateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, email_verified, password_hash, hash_type, name, avatar_url, mfa_enabled, mfa_secret, mfa_verified, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, boolToInt(u.EmailVerified), u.PasswordHash, u.HashType,
		u.Name, u.AvatarURL, boolToInt(u.MFAEnabled), u.MFASecret, boolToInt(u.MFAVerified),
		u.Metadata, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, email_verified, password_hash, hash_type, name, avatar_url, mfa_enabled, mfa_secret, mfa_verified, metadata, created_at, updated_at
		 FROM users WHERE id = ?`, id))
}

func (s *SQLiteStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, email_verified, password_hash, hash_type, name, avatar_url, mfa_enabled, mfa_secret, mfa_verified, metadata, created_at, updated_at
		 FROM users WHERE email = ?`, email))
}

func (s *SQLiteStore) ListUsers(ctx context.Context, opts ListUsersOpts) ([]*User, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}

	var args []interface{}
	query := `SELECT id, email, email_verified, password_hash, hash_type, name, avatar_url, mfa_enabled, mfa_secret, mfa_verified, metadata, created_at, updated_at FROM users`

	if opts.Search != "" {
		query += ` WHERE email LIKE ? OR name LIKE ?`
		search := "%" + opts.Search + "%"
		args = append(args, search, search)
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, opts.Limit, opts.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := s.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *SQLiteStore) UpdateUser(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET email=?, email_verified=?, password_hash=?, hash_type=?, name=?, avatar_url=?, mfa_enabled=?, mfa_secret=?, mfa_verified=?, metadata=?, updated_at=?
		 WHERE id=?`,
		u.Email, boolToInt(u.EmailVerified), u.PasswordHash, u.HashType,
		u.Name, u.AvatarURL, boolToInt(u.MFAEnabled), u.MFASecret, boolToInt(u.MFAVerified),
		u.Metadata, u.UpdatedAt, u.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) scanUser(row *sql.Row) (*User, error) {
	var u User
	var emailVerified, mfaEnabled, mfaVerified int
	err := row.Scan(
		&u.ID, &u.Email, &emailVerified, &u.PasswordHash, &u.HashType,
		&u.Name, &u.AvatarURL, &mfaEnabled, &u.MFASecret, &mfaVerified,
		&u.Metadata, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.EmailVerified = emailVerified != 0
	u.MFAEnabled = mfaEnabled != 0
	u.MFAVerified = mfaVerified != 0
	return &u, nil
}

func (s *SQLiteStore) scanUserFromRows(rows *sql.Rows) (*User, error) {
	var u User
	var emailVerified, mfaEnabled, mfaVerified int
	err := rows.Scan(
		&u.ID, &u.Email, &emailVerified, &u.PasswordHash, &u.HashType,
		&u.Name, &u.AvatarURL, &mfaEnabled, &u.MFASecret, &mfaVerified,
		&u.Metadata, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.EmailVerified = emailVerified != 0
	u.MFAEnabled = mfaEnabled != 0
	u.MFAVerified = mfaVerified != 0
	return &u, nil
}

// --- Sessions ---

func (s *SQLiteStore) CreateSession(ctx context.Context, sess *Session) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, ip, user_agent, mfa_passed, auth_method, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.UserID, sess.IP, sess.UserAgent,
		boolToInt(sess.MFAPassed), sess.AuthMethod, sess.ExpiresAt, sess.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetSessionByID(ctx context.Context, id string) (*Session, error) {
	var sess Session
	var mfaPassed int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, ip, user_agent, mfa_passed, auth_method, expires_at, created_at
		 FROM sessions WHERE id = ?`, id,
	).Scan(&sess.ID, &sess.UserID, &sess.IP, &sess.UserAgent, &mfaPassed,
		&sess.AuthMethod, &sess.ExpiresAt, &sess.CreatedAt)
	if err != nil {
		return nil, err
	}
	sess.MFAPassed = mfaPassed != 0
	return &sess, nil
}

func (s *SQLiteStore) GetSessionsByUserID(ctx context.Context, userID string) ([]*Session, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, ip, user_agent, mfa_passed, auth_method, expires_at, created_at
		 FROM sessions WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var sess Session
		var mfaPassed int
		if err := rows.Scan(&sess.ID, &sess.UserID, &sess.IP, &sess.UserAgent, &mfaPassed,
			&sess.AuthMethod, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
			return nil, err
		}
		sess.MFAPassed = mfaPassed != 0
		sessions = append(sessions, &sess)
	}
	return sessions, rows.Err()
}

func (s *SQLiteStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) UpdateSessionMFAPassed(ctx context.Context, id string, mfaPassed bool) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET mfa_passed = ? WHERE id = ?`,
		boolToInt(mfaPassed), id,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// --- OAuthAccounts ---

func (s *SQLiteStore) CreateOAuthAccount(ctx context.Context, a *OAuthAccount) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO oauth_accounts (id, user_id, provider, provider_id, email, access_token, refresh_token, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.UserID, a.Provider, a.ProviderID, a.Email, a.AccessToken, a.RefreshToken, a.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetOAuthAccountByProviderID(ctx context.Context, provider, providerID string) (*OAuthAccount, error) {
	var a OAuthAccount
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, provider, provider_id, email, access_token, refresh_token, created_at
		 FROM oauth_accounts WHERE provider = ? AND provider_id = ?`, provider, providerID,
	).Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderID, &a.Email, &a.AccessToken, &a.RefreshToken, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *SQLiteStore) GetOAuthAccountsByUserID(ctx context.Context, userID string) ([]*OAuthAccount, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, provider, provider_id, email, access_token, refresh_token, created_at
		 FROM oauth_accounts WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*OAuthAccount
	for rows.Next() {
		var a OAuthAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.Provider, &a.ProviderID, &a.Email, &a.AccessToken, &a.RefreshToken, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}
	return accounts, rows.Err()
}

func (s *SQLiteStore) DeleteOAuthAccount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oauth_accounts WHERE id = ?`, id)
	return err
}

// --- PasskeyCredentials ---

func (s *SQLiteStore) CreatePasskeyCredential(ctx context.Context, c *PasskeyCredential) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO passkey_credentials (id, user_id, credential_id, public_key, aaguid, sign_count, name, transports, backed_up, created_at, last_used_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.UserID, c.CredentialID, c.PublicKey, c.AAGUID, c.SignCount,
		c.Name, c.Transports, boolToInt(c.BackedUp), c.CreatedAt, c.LastUsedAt,
	)
	return err
}

func (s *SQLiteStore) GetPasskeyByCredentialID(ctx context.Context, credentialID []byte) (*PasskeyCredential, error) {
	var c PasskeyCredential
	var backedUp int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, credential_id, public_key, aaguid, sign_count, name, transports, backed_up, created_at, last_used_at
		 FROM passkey_credentials WHERE credential_id = ?`, credentialID,
	).Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.AAGUID, &c.SignCount,
		&c.Name, &c.Transports, &backedUp, &c.CreatedAt, &c.LastUsedAt)
	if err != nil {
		return nil, err
	}
	c.BackedUp = backedUp != 0
	return &c, nil
}

func (s *SQLiteStore) GetPasskeysByUserID(ctx context.Context, userID string) ([]*PasskeyCredential, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, credential_id, public_key, aaguid, sign_count, name, transports, backed_up, created_at, last_used_at
		 FROM passkey_credentials WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []*PasskeyCredential
	for rows.Next() {
		var c PasskeyCredential
		var backedUp int
		if err := rows.Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &c.AAGUID, &c.SignCount,
			&c.Name, &c.Transports, &backedUp, &c.CreatedAt, &c.LastUsedAt); err != nil {
			return nil, err
		}
		c.BackedUp = backedUp != 0
		creds = append(creds, &c)
	}
	return creds, rows.Err()
}

func (s *SQLiteStore) UpdatePasskeyCredential(ctx context.Context, c *PasskeyCredential) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE passkey_credentials SET sign_count=?, name=?, last_used_at=? WHERE id=?`,
		c.SignCount, c.Name, c.LastUsedAt, c.ID,
	)
	return err
}

func (s *SQLiteStore) DeletePasskeyCredential(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM passkey_credentials WHERE id = ?`, id)
	return err
}

// --- MagicLinkTokens ---

func (s *SQLiteStore) CreateMagicLinkToken(ctx context.Context, t *MagicLinkToken) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO magic_link_tokens (id, email, token_hash, used, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.Email, t.TokenHash, boolToInt(t.Used), t.ExpiresAt, t.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetMagicLinkTokenByHash(ctx context.Context, tokenHash string) (*MagicLinkToken, error) {
	var t MagicLinkToken
	var used int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, token_hash, used, expires_at, created_at
		 FROM magic_link_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&t.ID, &t.Email, &t.TokenHash, &used, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	t.Used = used != 0
	return &t, nil
}

func (s *SQLiteStore) MarkMagicLinkTokenUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE magic_link_tokens SET used = 1 WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) DeleteExpiredMagicLinkTokens(ctx context.Context) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `DELETE FROM magic_link_tokens WHERE expires_at < ?`, now)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// --- MFARecoveryCodes ---

func (s *SQLiteStore) CreateMFARecoveryCodes(ctx context.Context, codes []*MFARecoveryCode) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO mfa_recovery_codes (id, user_id, code, used, created_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range codes {
		if _, err := stmt.ExecContext(ctx, c.ID, c.UserID, c.Code, boolToInt(c.Used), c.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetMFARecoveryCodesByUserID(ctx context.Context, userID string) ([]*MFARecoveryCode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, code, used, created_at FROM mfa_recovery_codes WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []*MFARecoveryCode
	for rows.Next() {
		var c MFARecoveryCode
		var used int
		if err := rows.Scan(&c.ID, &c.UserID, &c.Code, &used, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Used = used != 0
		codes = append(codes, &c)
	}
	return codes, rows.Err()
}

func (s *SQLiteStore) MarkMFARecoveryCodeUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE mfa_recovery_codes SET used = 1 WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) DeleteAllMFARecoveryCodesByUserID(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = ?`, userID)
	return err
}

// --- Roles ---

func (s *SQLiteStore) CreateRole(ctx context.Context, r *Role) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO roles (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.Description, r.CreatedAt, r.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetRoleByID(ctx context.Context, id string) (*Role, error) {
	var r Role
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM roles WHERE id = ?`, id,
	).Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *SQLiteStore) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	var r Role
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM roles WHERE name = ?`, name,
	).Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *SQLiteStore) ListRoles(ctx context.Context) ([]*Role, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, &r)
	}
	return roles, rows.Err()
}

func (s *SQLiteStore) UpdateRole(ctx context.Context, r *Role) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE roles SET name=?, description=?, updated_at=? WHERE id=?`,
		r.Name, r.Description, r.UpdatedAt, r.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteRole(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM roles WHERE id = ?`, id)
	return err
}

// --- Permissions ---

func (s *SQLiteStore) CreatePermission(ctx context.Context, p *Permission) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO permissions (id, action, resource, created_at) VALUES (?, ?, ?, ?)`,
		p.ID, p.Action, p.Resource, p.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetPermissionByID(ctx context.Context, id string) (*Permission, error) {
	var p Permission
	err := s.db.QueryRowContext(ctx,
		`SELECT id, action, resource, created_at FROM permissions WHERE id = ?`, id,
	).Scan(&p.ID, &p.Action, &p.Resource, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *SQLiteStore) ListPermissions(ctx context.Context) ([]*Permission, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, action, resource, created_at FROM permissions ORDER BY resource, action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Action, &p.Resource, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, &p)
	}
	return perms, rows.Err()
}

func (s *SQLiteStore) GetPermissionByActionResource(ctx context.Context, action, resource string) (*Permission, error) {
	var p Permission
	err := s.db.QueryRowContext(ctx,
		`SELECT id, action, resource, created_at FROM permissions WHERE action = ? AND resource = ?`, action, resource,
	).Scan(&p.ID, &p.Action, &p.Resource, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// --- RolePermissions ---

func (s *SQLiteStore) AttachPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO role_permissions (role_id, permission_id) VALUES (?, ?)`,
		roleID, permissionID,
	)
	return err
}

func (s *SQLiteStore) DetachPermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM role_permissions WHERE role_id = ? AND permission_id = ?`,
		roleID, permissionID,
	)
	return err
}

func (s *SQLiteStore) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*Permission, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.id, p.action, p.resource, p.created_at
		 FROM permissions p
		 INNER JOIN role_permissions rp ON rp.permission_id = p.id
		 WHERE rp.role_id = ?
		 ORDER BY p.resource, p.action`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Action, &p.Resource, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, &p)
	}
	return perms, rows.Err()
}

// --- UserRoles ---

func (s *SQLiteStore) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO user_roles (user_id, role_id) VALUES (?, ?)`,
		userID, roleID,
	)
	return err
}

func (s *SQLiteStore) RemoveRoleFromUser(ctx context.Context, userID, roleID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM user_roles WHERE user_id = ? AND role_id = ?`,
		userID, roleID,
	)
	return err
}

func (s *SQLiteStore) GetRolesByUserID(ctx context.Context, userID string) ([]*Role, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		 FROM roles r
		 INNER JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = ?
		 ORDER BY r.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, &r)
	}
	return roles, rows.Err()
}

func (s *SQLiteStore) GetUsersByRoleID(ctx context.Context, roleID string) ([]*User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.email, u.email_verified, u.password_hash, u.hash_type, u.name, u.avatar_url, u.mfa_enabled, u.mfa_secret, u.mfa_verified, u.metadata, u.created_at, u.updated_at
		 FROM users u
		 INNER JOIN user_roles ur ON ur.user_id = u.id
		 WHERE ur.role_id = ?`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := s.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// --- SSOConnections ---

func (s *SQLiteStore) CreateSSOConnection(ctx context.Context, c *SSOConnection) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sso_connections (id, type, name, domain, saml_idp_url, saml_idp_cert, saml_sp_entity_id, saml_sp_acs_url, oidc_issuer, oidc_client_id, oidc_client_secret, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Type, c.Name, c.Domain,
		c.SAMLIdPURL, c.SAMLIdPCert, c.SAMLSPEntityID, c.SAMLSPAcsURL,
		c.OIDCIssuer, c.OIDCClientID, c.OIDCClientSecret,
		boolToInt(c.Enabled), c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetSSOConnectionByID(ctx context.Context, id string) (*SSOConnection, error) {
	return s.scanSSOConnection(s.db.QueryRowContext(ctx,
		`SELECT id, type, name, domain, saml_idp_url, saml_idp_cert, saml_sp_entity_id, saml_sp_acs_url, oidc_issuer, oidc_client_id, oidc_client_secret, enabled, created_at, updated_at
		 FROM sso_connections WHERE id = ?`, id))
}

func (s *SQLiteStore) GetSSOConnectionByDomain(ctx context.Context, domain string) (*SSOConnection, error) {
	return s.scanSSOConnection(s.db.QueryRowContext(ctx,
		`SELECT id, type, name, domain, saml_idp_url, saml_idp_cert, saml_sp_entity_id, saml_sp_acs_url, oidc_issuer, oidc_client_id, oidc_client_secret, enabled, created_at, updated_at
		 FROM sso_connections WHERE domain = ? AND enabled = 1`, domain))
}

func (s *SQLiteStore) ListSSOConnections(ctx context.Context) ([]*SSOConnection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, name, domain, saml_idp_url, saml_idp_cert, saml_sp_entity_id, saml_sp_acs_url, oidc_issuer, oidc_client_id, oidc_client_secret, enabled, created_at, updated_at
		 FROM sso_connections ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []*SSOConnection
	for rows.Next() {
		var c SSOConnection
		var enabled int
		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Domain,
			&c.SAMLIdPURL, &c.SAMLIdPCert, &c.SAMLSPEntityID, &c.SAMLSPAcsURL,
			&c.OIDCIssuer, &c.OIDCClientID, &c.OIDCClientSecret,
			&enabled, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Enabled = enabled != 0
		conns = append(conns, &c)
	}
	return conns, rows.Err()
}

func (s *SQLiteStore) UpdateSSOConnection(ctx context.Context, c *SSOConnection) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sso_connections SET type=?, name=?, domain=?, saml_idp_url=?, saml_idp_cert=?, saml_sp_entity_id=?, saml_sp_acs_url=?, oidc_issuer=?, oidc_client_id=?, oidc_client_secret=?, enabled=?, updated_at=?
		 WHERE id=?`,
		c.Type, c.Name, c.Domain,
		c.SAMLIdPURL, c.SAMLIdPCert, c.SAMLSPEntityID, c.SAMLSPAcsURL,
		c.OIDCIssuer, c.OIDCClientID, c.OIDCClientSecret,
		boolToInt(c.Enabled), c.UpdatedAt, c.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteSSOConnection(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sso_connections WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) scanSSOConnection(row *sql.Row) (*SSOConnection, error) {
	var c SSOConnection
	var enabled int
	err := row.Scan(&c.ID, &c.Type, &c.Name, &c.Domain,
		&c.SAMLIdPURL, &c.SAMLIdPCert, &c.SAMLSPEntityID, &c.SAMLSPAcsURL,
		&c.OIDCIssuer, &c.OIDCClientID, &c.OIDCClientSecret,
		&enabled, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c.Enabled = enabled != 0
	return &c, nil
}

// --- SSOIdentities ---

func (s *SQLiteStore) CreateSSOIdentity(ctx context.Context, i *SSOIdentity) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sso_identities (id, user_id, connection_id, provider_sub, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		i.ID, i.UserID, i.ConnectionID, i.ProviderSub, i.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetSSOIdentityByConnectionAndSub(ctx context.Context, connectionID, providerSub string) (*SSOIdentity, error) {
	var i SSOIdentity
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, connection_id, provider_sub, created_at
		 FROM sso_identities WHERE connection_id = ? AND provider_sub = ?`,
		connectionID, providerSub,
	).Scan(&i.ID, &i.UserID, &i.ConnectionID, &i.ProviderSub, &i.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (s *SQLiteStore) GetSSOIdentitiesByUserID(ctx context.Context, userID string) ([]*SSOIdentity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, connection_id, provider_sub, created_at
		 FROM sso_identities WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var idents []*SSOIdentity
	for rows.Next() {
		var i SSOIdentity
		if err := rows.Scan(&i.ID, &i.UserID, &i.ConnectionID, &i.ProviderSub, &i.CreatedAt); err != nil {
			return nil, err
		}
		idents = append(idents, &i)
	}
	return idents, rows.Err()
}

// --- APIKeys ---

func (s *SQLiteStore) CreateAPIKey(ctx context.Context, k *APIKey) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO api_keys (id, name, key_hash, key_prefix, key_suffix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.Name, k.KeyHash, k.KeyPrefix, k.KeySuffix, k.Scopes, k.RateLimit,
		k.ExpiresAt, k.LastUsedAt, k.CreatedAt, k.RevokedAt,
	)
	return err
}

func (s *SQLiteStore) GetAPIKeyByKeyHash(ctx context.Context, keyHash string) (*APIKey, error) {
	return s.scanAPIKey(s.db.QueryRowContext(ctx,
		`SELECT id, name, key_hash, key_prefix, key_suffix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		 FROM api_keys WHERE key_hash = ?`, keyHash))
}

func (s *SQLiteStore) GetAPIKeyByID(ctx context.Context, id string) (*APIKey, error) {
	return s.scanAPIKey(s.db.QueryRowContext(ctx,
		`SELECT id, name, key_hash, key_prefix, key_suffix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		 FROM api_keys WHERE id = ?`, id))
}

func (s *SQLiteStore) ListAPIKeys(ctx context.Context) ([]*APIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, key_hash, key_prefix, key_suffix, scopes, rate_limit, expires_at, last_used_at, created_at, revoked_at
		 FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.KeySuffix, &k.Scopes, &k.RateLimit,
			&k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}
	return keys, rows.Err()
}

func (s *SQLiteStore) UpdateAPIKey(ctx context.Context, k *APIKey) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET name=?, scopes=?, rate_limit=?, expires_at=?, last_used_at=? WHERE id=?`,
		k.Name, k.Scopes, k.RateLimit, k.ExpiresAt, k.LastUsedAt, k.ID,
	)
	return err
}

func (s *SQLiteStore) RevokeAPIKey(ctx context.Context, id string, revokedAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET revoked_at = ? WHERE id = ?`,
		revokedAt.UTC().Format(time.RFC3339), id,
	)
	return err
}

func (s *SQLiteStore) scanAPIKey(row *sql.Row) (*APIKey, error) {
	var k APIKey
	err := row.Scan(&k.ID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.KeySuffix, &k.Scopes, &k.RateLimit,
		&k.ExpiresAt, &k.LastUsedAt, &k.CreatedAt, &k.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (s *SQLiteStore) CountActiveAPIKeysByScope(ctx context.Context, scope string) (int, error) {
	var count int
	pattern := fmt.Sprintf("%%%q%%", scope)
	// Match keys that contain the scope in their JSON scopes array, are not revoked, and not expired
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM api_keys
		 WHERE revoked_at IS NULL
		 AND (expires_at IS NULL OR expires_at = '' OR expires_at > ?)
		 AND (scopes LIKE ? OR scopes LIKE '%"*"%')`,
		time.Now().UTC().Format(time.RFC3339), pattern,
	).Scan(&count)
	return count, err
}

// --- AuditLogs ---

func (s *SQLiteStore) CreateAuditLog(ctx context.Context, l *AuditLog) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_logs (id, actor_id, actor_type, action, target_type, target_id, ip, user_agent, metadata, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.ActorID, l.ActorType, l.Action, l.TargetType, l.TargetID,
		l.IP, l.UserAgent, l.Metadata, l.Status, l.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) GetAuditLogByID(ctx context.Context, id string) (*AuditLog, error) {
	var l AuditLog
	err := s.db.QueryRowContext(ctx,
		`SELECT id, actor_id, actor_type, action, target_type, target_id, ip, user_agent, metadata, status, created_at
		 FROM audit_logs WHERE id = ?`, id,
	).Scan(&l.ID, &l.ActorID, &l.ActorType, &l.Action, &l.TargetType, &l.TargetID,
		&l.IP, &l.UserAgent, &l.Metadata, &l.Status, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *SQLiteStore) QueryAuditLogs(ctx context.Context, opts AuditLogQuery) ([]*AuditLog, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}

	var conditions []string
	var args []interface{}

	if opts.Action != "" {
		// Support comma-separated actions
		actions := strings.Split(opts.Action, ",")
		placeholders := make([]string, len(actions))
		for i, a := range actions {
			placeholders[i] = "?"
			args = append(args, strings.TrimSpace(a))
		}
		conditions = append(conditions, fmt.Sprintf("action IN (%s)", strings.Join(placeholders, ",")))
	}
	if opts.ActorID != "" {
		conditions = append(conditions, "actor_id = ?")
		args = append(args, opts.ActorID)
	}
	if opts.TargetID != "" {
		conditions = append(conditions, "target_id = ?")
		args = append(args, opts.TargetID)
	}
	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}
	if opts.IP != "" {
		conditions = append(conditions, "ip = ?")
		args = append(args, opts.IP)
	}
	if opts.From != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, opts.From)
	}
	if opts.To != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, opts.To)
	}
	if opts.Cursor != "" {
		// Cursor-based pagination: get items with created_at before the cursor item
		conditions = append(conditions, "created_at <= (SELECT created_at FROM audit_logs WHERE id = ?)")
		conditions = append(conditions, "id != ?")
		args = append(args, opts.Cursor, opts.Cursor)
	}

	query := "SELECT id, actor_id, actor_type, action, target_type, target_id, ip, user_agent, metadata, status, created_at FROM audit_logs"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorType, &l.Action, &l.TargetType, &l.TargetID,
			&l.IP, &l.UserAgent, &l.Metadata, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, rows.Err()
}

func (s *SQLiteStore) DeleteAuditLogsBefore(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM audit_logs WHERE created_at < ?`,
		before.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// --- Migrations ---

func (s *SQLiteStore) CreateMigration(ctx context.Context, m *Migration) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO migrations (id, source, status, users_total, users_imported, errors, created_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Source, m.Status, m.UsersTotal, m.UsersImported, m.Errors, m.CreatedAt, m.CompletedAt,
	)
	return err
}

func (s *SQLiteStore) GetMigrationByID(ctx context.Context, id string) (*Migration, error) {
	var m Migration
	err := s.db.QueryRowContext(ctx,
		`SELECT id, source, status, users_total, users_imported, errors, created_at, completed_at
		 FROM migrations WHERE id = ?`, id,
	).Scan(&m.ID, &m.Source, &m.Status, &m.UsersTotal, &m.UsersImported, &m.Errors, &m.CreatedAt, &m.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *SQLiteStore) ListMigrations(ctx context.Context) ([]*Migration, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source, status, users_total, users_imported, errors, created_at, completed_at
		 FROM migrations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []*Migration
	for rows.Next() {
		var m Migration
		if err := rows.Scan(&m.ID, &m.Source, &m.Status, &m.UsersTotal, &m.UsersImported, &m.Errors, &m.CreatedAt, &m.CompletedAt); err != nil {
			return nil, err
		}
		migrations = append(migrations, &m)
	}
	return migrations, rows.Err()
}

func (s *SQLiteStore) UpdateMigration(ctx context.Context, m *Migration) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE migrations SET status=?, users_total=?, users_imported=?, errors=?, completed_at=? WHERE id=?`,
		m.Status, m.UsersTotal, m.UsersImported, m.Errors, m.CompletedAt, m.ID,
	)
	return err
}

// --- Stats / metrics ---

func (s *SQLiteStore) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *SQLiteStore) CountUsersCreatedSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE created_at >= ?`,
		since.UTC().Format(time.RFC3339),
	).Scan(&n)
	return n, err
}

func (s *SQLiteStore) CountActiveSessions(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE expires_at > ?`,
		time.Now().UTC().Format(time.RFC3339),
	).Scan(&n)
	return n, err
}

func (s *SQLiteStore) CountMFAEnabled(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE mfa_enabled = 1`).Scan(&n)
	return n, err
}

func (s *SQLiteStore) CountFailedLoginsSince(ctx context.Context, since time.Time) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE action = 'login' AND status = 'failure' AND created_at >= ?`,
		since.UTC().Format(time.RFC3339),
	).Scan(&n)
	return n, err
}

// CountExpiringAPIKeys counts active (not revoked) keys that expire within the given window.
// Keys with NULL expires_at (never expire) are excluded.
func (s *SQLiteStore) CountExpiringAPIKeys(ctx context.Context, within time.Duration) (int, error) {
	now := time.Now().UTC()
	cutoff := now.Add(within).Format(time.RFC3339)
	nowStr := now.Format(time.RFC3339)
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM api_keys
		 WHERE revoked_at IS NULL
		   AND expires_at IS NOT NULL
		   AND expires_at > ?
		   AND expires_at <= ?`,
		nowStr, cutoff,
	).Scan(&n)
	return n, err
}

func (s *SQLiteStore) CountSSOConnections(ctx context.Context, enabledOnly bool) (int, error) {
	q := `SELECT COUNT(*) FROM sso_connections`
	if enabledOnly {
		q += ` WHERE enabled = 1`
	}
	var n int
	err := s.db.QueryRowContext(ctx, q).Scan(&n)
	return n, err
}

func (s *SQLiteStore) GroupSessionsByAuthMethodSince(ctx context.Context, since time.Time) ([]MethodCount, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT auth_method, COUNT(*) FROM sessions
		 WHERE created_at >= ?
		 GROUP BY auth_method ORDER BY COUNT(*) DESC`,
		since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MethodCount
	for rows.Next() {
		var mc MethodCount
		if err := rows.Scan(&mc.AuthMethod, &mc.Count); err != nil {
			return nil, err
		}
		out = append(out, mc)
	}
	return out, rows.Err()
}

// GroupUsersCreatedByDay returns a per-day signup count for the last N days.
// Days with zero signups are OMITTED; callers are responsible for filling gaps
// (makes the query index-friendly at any scale).
func (s *SQLiteStore) GroupUsersCreatedByDay(ctx context.Context, days int) ([]DayCount, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx,
		`SELECT substr(created_at, 1, 10) AS day, COUNT(*) FROM users
		 WHERE created_at >= ?
		 GROUP BY day ORDER BY day ASC`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DayCount
	for rows.Next() {
		var d DayCount
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// --- Admin session listing ---

// ListActiveSessions returns active sessions joined with user email, filtered + cursor-paginated.
// Keyset cursor format: "<created_at>|<id>". Uses (created_at DESC, id DESC) for stable order.
func (s *SQLiteStore) ListActiveSessions(ctx context.Context, opts ListSessionsOpts) ([]*SessionWithUser, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var (
		where []string
		args  []interface{}
	)
	where = append(where, `s.expires_at > ?`)
	args = append(args, time.Now().UTC().Format(time.RFC3339))

	if opts.UserID != "" {
		where = append(where, `s.user_id = ?`)
		args = append(args, opts.UserID)
	}
	if opts.AuthMethod != "" {
		where = append(where, `s.auth_method = ?`)
		args = append(args, opts.AuthMethod)
	}
	if opts.MFAPassed != nil {
		where = append(where, `s.mfa_passed = ?`)
		args = append(args, boolToInt(*opts.MFAPassed))
	}

	if opts.Cursor != "" {
		parts := strings.SplitN(opts.Cursor, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid cursor")
		}
		// Keyset: items strictly after the cursor tuple (created_at, id) DESC.
		// Equivalent to: created_at < cursor_created OR (= AND id < cursor_id).
		where = append(where, `(s.created_at < ? OR (s.created_at = ? AND s.id < ?))`)
		args = append(args, parts[0], parts[0], parts[1])
	}

	q := `SELECT s.id, s.user_id, s.ip, s.user_agent, s.mfa_passed, s.auth_method, s.expires_at, s.created_at,
	             COALESCE(u.email, '')
	      FROM sessions s
	      LEFT JOIN users u ON u.id = s.user_id
	      WHERE ` + strings.Join(where, " AND ") + `
	      ORDER BY s.created_at DESC, s.id DESC
	      LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*SessionWithUser
	for rows.Next() {
		var sw SessionWithUser
		var mfa int
		if err := rows.Scan(&sw.ID, &sw.UserID, &sw.IP, &sw.UserAgent, &mfa,
			&sw.AuthMethod, &sw.ExpiresAt, &sw.CreatedAt, &sw.UserEmail); err != nil {
			return nil, err
		}
		sw.MFAPassed = mfa != 0
		out = append(out, &sw)
	}
	return out, rows.Err()
}

// DeleteSessionsByUserID deletes every session for a user and returns the IDs
// of the deleted sessions so the caller can emit one audit entry per session.
func (s *SQLiteStore) DeleteSessionsByUserID(ctx context.Context, userID string) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `SELECT id FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

// --- Dev inbox ---

func (s *SQLiteStore) CreateDevEmail(ctx context.Context, e *DevEmail) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO dev_emails (id, to_addr, subject, html, text, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.To, e.Subject, e.HTML, e.Text, e.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) ListDevEmails(ctx context.Context, limit int) ([]*DevEmail, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, to_addr, subject, html, text, created_at
		 FROM dev_emails ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*DevEmail
	for rows.Next() {
		var e DevEmail
		if err := rows.Scan(&e.ID, &e.To, &e.Subject, &e.HTML, &e.Text, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetDevEmail(ctx context.Context, id string) (*DevEmail, error) {
	var e DevEmail
	err := s.db.QueryRowContext(ctx,
		`SELECT id, to_addr, subject, html, text, created_at FROM dev_emails WHERE id = ?`,
		id,
	).Scan(&e.ID, &e.To, &e.Subject, &e.HTML, &e.Text, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *SQLiteStore) DeleteAllDevEmails(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dev_emails`)
	return err
}

// --- Helpers ---

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
