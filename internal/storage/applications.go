package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// --- Applications ---

func (s *SQLiteStore) CreateApplication(ctx context.Context, app *Application) error {
	callbackJSON, err := marshalStringSlice(app.AllowedCallbackURLs)
	if err != nil {
		return fmt.Errorf("marshal allowed_callback_urls: %w", err)
	}
	logoutJSON, err := marshalStringSlice(app.AllowedLogoutURLs)
	if err != nil {
		return fmt.Errorf("marshal allowed_logout_urls: %w", err)
	}
	originsJSON, err := marshalStringSlice(app.AllowedOrigins)
	if err != nil {
		return fmt.Errorf("marshal allowed_origins: %w", err)
	}
	metaJSON, err := marshalMetadata(app.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	integrationMode := app.IntegrationMode
	if integrationMode == "" {
		integrationMode = "custom"
	}
	proxyFallback := app.ProxyLoginFallback
	if proxyFallback == "" {
		proxyFallback = "hosted"
	}
	var brandingOverride sql.NullString
	if app.BrandingOverride != "" {
		brandingOverride = sql.NullString{String: app.BrandingOverride, Valid: true}
	}
	var proxyFallbackURL sql.NullString
	if app.ProxyLoginFallbackURL != "" {
		proxyFallbackURL = sql.NullString{String: app.ProxyLoginFallbackURL, Valid: true}
	}

	var slugVal sql.NullString
	if app.Slug != "" {
		slugVal = sql.NullString{String: app.Slug, Valid: true}
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO applications
		 (id, name, slug, client_id, client_secret_hash, client_secret_prefix,
		  allowed_callback_urls, allowed_logout_urls, allowed_origins,
		  is_default, metadata, created_at, updated_at,
		  integration_mode, branding_override, proxy_login_fallback, proxy_login_fallback_url)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		app.ID, app.Name, slugVal, app.ClientID, app.ClientSecretHash, app.ClientSecretPrefix,
		callbackJSON, logoutJSON, originsJSON,
		boolToInt(app.IsDefault), metaJSON,
		app.CreatedAt.UTC().Format(time.RFC3339),
		app.UpdatedAt.UTC().Format(time.RFC3339),
		integrationMode, brandingOverride, proxyFallback, proxyFallbackURL,
	)
	return err
}

func (s *SQLiteStore) GetApplicationByID(ctx context.Context, id string) (*Application, error) {
	return s.scanApplication(s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, client_id, client_secret_hash, client_secret_prefix,
		        allowed_callback_urls, allowed_logout_urls, allowed_origins,
		        is_default, metadata, created_at, updated_at,
		        integration_mode, branding_override, proxy_login_fallback, proxy_login_fallback_url
		 FROM applications WHERE id = ?`, id))
}

func (s *SQLiteStore) GetApplicationByClientID(ctx context.Context, clientID string) (*Application, error) {
	return s.scanApplication(s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, client_id, client_secret_hash, client_secret_prefix,
		        allowed_callback_urls, allowed_logout_urls, allowed_origins,
		        is_default, metadata, created_at, updated_at,
		        integration_mode, branding_override, proxy_login_fallback, proxy_login_fallback_url
		 FROM applications WHERE client_id = ?`, clientID))
}

// GetApplicationBySlug returns the application with the given slug.
// Returns sql.ErrNoRows when no matching row exists.
func (s *SQLiteStore) GetApplicationBySlug(ctx context.Context, slug string) (*Application, error) {
	return s.scanApplication(s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, client_id, client_secret_hash, client_secret_prefix,
		        allowed_callback_urls, allowed_logout_urls, allowed_origins,
		        is_default, metadata, created_at, updated_at,
		        integration_mode, branding_override, proxy_login_fallback, proxy_login_fallback_url
		 FROM applications WHERE slug = ?`, slug))
}

func (s *SQLiteStore) GetDefaultApplication(ctx context.Context) (*Application, error) {
	return s.scanApplication(s.db.QueryRowContext(ctx,
		`SELECT id, name, slug, client_id, client_secret_hash, client_secret_prefix,
		        allowed_callback_urls, allowed_logout_urls, allowed_origins,
		        is_default, metadata, created_at, updated_at,
		        integration_mode, branding_override, proxy_login_fallback, proxy_login_fallback_url
		 FROM applications WHERE is_default = 1 LIMIT 1`))
}

func (s *SQLiteStore) ListApplications(ctx context.Context, limit, offset int) ([]*Application, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, slug, client_id, client_secret_hash, client_secret_prefix,
		        allowed_callback_urls, allowed_logout_urls, allowed_origins,
		        is_default, metadata, created_at, updated_at,
		        integration_mode, branding_override, proxy_login_fallback, proxy_login_fallback_url
		 FROM applications ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Application
	for rows.Next() {
		app, err := s.scanApplicationFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateApplication(ctx context.Context, app *Application) error {
	callbackJSON, err := marshalStringSlice(app.AllowedCallbackURLs)
	if err != nil {
		return fmt.Errorf("marshal allowed_callback_urls: %w", err)
	}
	logoutJSON, err := marshalStringSlice(app.AllowedLogoutURLs)
	if err != nil {
		return fmt.Errorf("marshal allowed_logout_urls: %w", err)
	}
	originsJSON, err := marshalStringSlice(app.AllowedOrigins)
	if err != nil {
		return fmt.Errorf("marshal allowed_origins: %w", err)
	}
	metaJSON, err := marshalMetadata(app.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	integrationMode := app.IntegrationMode
	if integrationMode == "" {
		integrationMode = "custom"
	}
	proxyFallback := app.ProxyLoginFallback
	if proxyFallback == "" {
		proxyFallback = "hosted"
	}
	var brandingOverride sql.NullString
	if app.BrandingOverride != "" {
		brandingOverride = sql.NullString{String: app.BrandingOverride, Valid: true}
	}
	var proxyFallbackURL sql.NullString
	if app.ProxyLoginFallbackURL != "" {
		proxyFallbackURL = sql.NullString{String: app.ProxyLoginFallbackURL, Valid: true}
	}

	var slugVal sql.NullString
	if app.Slug != "" {
		slugVal = sql.NullString{String: app.Slug, Valid: true}
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE applications SET
		   name = ?, slug = ?, allowed_callback_urls = ?, allowed_logout_urls = ?,
		   allowed_origins = ?, is_default = ?, metadata = ?,
		   integration_mode = ?, branding_override = ?,
		   proxy_login_fallback = ?, proxy_login_fallback_url = ?,
		   updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		app.Name, slugVal, callbackJSON, logoutJSON, originsJSON,
		boolToInt(app.IsDefault), metaJSON,
		integrationMode, brandingOverride, proxyFallback, proxyFallbackURL,
		app.ID,
	)
	return err
}

func (s *SQLiteStore) RotateApplicationSecret(ctx context.Context, id, newHash, newPrefix string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE applications SET
		   client_secret_hash = ?, client_secret_prefix = ?,
		   updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		newHash, newPrefix, id,
	)
	return err
}

func (s *SQLiteStore) DeleteApplication(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM applications WHERE id = ?`, id)
	return err
}

// scanApplication scans a single application row from a *sql.Row.
// Returns sql.ErrNoRows when the row does not exist (callers may compare to ErrNotFound).
func (s *SQLiteStore) scanApplication(row *sql.Row) (*Application, error) {
	var app Application
	var isDefault int
	var callbackJSON, logoutJSON, originsJSON, metaJSON string
	var createdAtStr, updatedAtStr string
	var integrationMode, proxyFallback string
	var slug, brandingOverride, proxyFallbackURL sql.NullString

	err := row.Scan(
		&app.ID, &app.Name, &slug, &app.ClientID, &app.ClientSecretHash, &app.ClientSecretPrefix,
		&callbackJSON, &logoutJSON, &originsJSON,
		&isDefault, &metaJSON, &createdAtStr, &updatedAtStr,
		&integrationMode, &brandingOverride, &proxyFallback, &proxyFallbackURL,
	)
	if err != nil {
		return nil, err
	}
	if slug.Valid {
		app.Slug = slug.String
	}
	app.IntegrationMode = integrationMode
	app.ProxyLoginFallback = proxyFallback
	if brandingOverride.Valid {
		app.BrandingOverride = brandingOverride.String
	}
	if proxyFallbackURL.Valid {
		app.ProxyLoginFallbackURL = proxyFallbackURL.String
	}
	return unmarshalApplication(&app, isDefault, callbackJSON, logoutJSON, originsJSON, metaJSON, createdAtStr, updatedAtStr)
}

// scanApplicationFromRows scans a single application row from *sql.Rows.
func (s *SQLiteStore) scanApplicationFromRows(rows *sql.Rows) (*Application, error) {
	var app Application
	var isDefault int
	var callbackJSON, logoutJSON, originsJSON, metaJSON string
	var createdAtStr, updatedAtStr string
	var integrationMode, proxyFallback string
	var slug, brandingOverride, proxyFallbackURL sql.NullString

	err := rows.Scan(
		&app.ID, &app.Name, &slug, &app.ClientID, &app.ClientSecretHash, &app.ClientSecretPrefix,
		&callbackJSON, &logoutJSON, &originsJSON,
		&isDefault, &metaJSON, &createdAtStr, &updatedAtStr,
		&integrationMode, &brandingOverride, &proxyFallback, &proxyFallbackURL,
	)
	if err != nil {
		return nil, err
	}
	if slug.Valid {
		app.Slug = slug.String
	}
	app.IntegrationMode = integrationMode
	app.ProxyLoginFallback = proxyFallback
	if brandingOverride.Valid {
		app.BrandingOverride = brandingOverride.String
	}
	if proxyFallbackURL.Valid {
		app.ProxyLoginFallbackURL = proxyFallbackURL.String
	}
	return unmarshalApplication(&app, isDefault, callbackJSON, logoutJSON, originsJSON, metaJSON, createdAtStr, updatedAtStr)
}

// unmarshalApplication fills in the JSON-column and time fields after a raw Scan.
func unmarshalApplication(
	app *Application,
	isDefault int,
	callbackJSON, logoutJSON, originsJSON, metaJSON string,
	createdAtStr, updatedAtStr string,
) (*Application, error) {
	app.IsDefault = isDefault != 0

	if err := json.Unmarshal([]byte(callbackJSON), &app.AllowedCallbackURLs); err != nil {
		return nil, fmt.Errorf("unmarshal allowed_callback_urls: %w", err)
	}
	if err := json.Unmarshal([]byte(logoutJSON), &app.AllowedLogoutURLs); err != nil {
		return nil, fmt.Errorf("unmarshal allowed_logout_urls: %w", err)
	}
	if err := json.Unmarshal([]byte(originsJSON), &app.AllowedOrigins); err != nil {
		return nil, fmt.Errorf("unmarshal allowed_origins: %w", err)
	}
	if err := json.Unmarshal([]byte(metaJSON), &app.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	// Normalise empty JSON arrays to non-nil slices so callers never deal with nil.
	if app.AllowedCallbackURLs == nil {
		app.AllowedCallbackURLs = []string{}
	}
	if app.AllowedLogoutURLs == nil {
		app.AllowedLogoutURLs = []string{}
	}
	if app.AllowedOrigins == nil {
		app.AllowedOrigins = []string{}
	}
	if app.Metadata == nil {
		app.Metadata = map[string]any{}
	}

	var err error
	app.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		// SQLite CURRENT_TIMESTAMP uses "YYYY-MM-DD HH:MM:SS" without the T separator.
		app.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse created_at %q: %w", createdAtStr, err)
		}
	}
	app.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		app.UpdatedAt, err = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse updated_at %q: %w", updatedAtStr, err)
		}
	}

	return app, nil
}

// --- JSON helpers ---

// marshalStringSlice encodes a string slice to JSON, always returning "[]" for nil/empty.
func marshalStringSlice(ss []string) (string, error) {
	if len(ss) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(ss)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// marshalMetadata encodes a map to JSON, always returning "{}" for nil/empty.
func marshalMetadata(m map[string]any) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
