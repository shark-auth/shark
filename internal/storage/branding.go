package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// GetBranding returns a branding row by id (e.g. "global").
func (s *SQLiteStore) GetBranding(ctx context.Context, id string) (*BrandingConfig, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(logo_url,''), COALESCE(logo_sha,''), primary_color, secondary_color,
		       font_family, footer_text, email_from_name, email_from_address
		FROM branding WHERE id = ?`, id)
	b := &BrandingConfig{}
	err := row.Scan(&b.LogoURL, &b.LogoSHA, &b.PrimaryColor, &b.SecondaryColor,
		&b.FontFamily, &b.FooterText, &b.EmailFromName, &b.EmailFromAddress)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("branding not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

// UpdateBranding updates allowed branding fields for the given id.
// Unknown field names are silently dropped.
func (s *SQLiteStore) UpdateBranding(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	allowed := map[string]bool{
		"primary_color": true, "secondary_color": true, "font_family": true,
		"footer_text": true, "email_from_name": true, "email_from_address": true,
	}
	setParts := []string{}
	args := []any{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		setParts = append(setParts, k+" = ?")
		args = append(args, v)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)
	q := "UPDATE branding SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

// SetBrandingLogo sets logo url + sha on the branding row.
func (s *SQLiteStore) SetBrandingLogo(ctx context.Context, id, url, sha string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE branding SET logo_url = ?, logo_sha = ?, updated_at = ? WHERE id = ?`,
		url, sha, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// ClearBrandingLogo nulls out logo url + sha.
func (s *SQLiteStore) ClearBrandingLogo(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE branding SET logo_url = NULL, logo_sha = NULL, updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// ResolveBranding returns global branding merged with per-app override.
// appID empty → global only. Missing override → global. Empty override fields fall through.
func (s *SQLiteStore) ResolveBranding(ctx context.Context, appID string) (*BrandingConfig, error) {
	global, err := s.GetBranding(ctx, "global")
	if err != nil {
		return nil, fmt.Errorf("resolve branding global: %w", err)
	}
	if appID == "" {
		return global, nil
	}

	var override sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT branding_override FROM applications WHERE id = ?`, appID).Scan(&override)
	if errors.Is(err, sql.ErrNoRows) || !override.Valid || override.String == "" {
		return global, nil
	}
	if err != nil {
		return global, nil
	}

	var over map[string]string
	if err := json.Unmarshal([]byte(override.String), &over); err != nil {
		return global, nil
	}

	merged := *global
	if v := over["logo_url"]; v != "" {
		merged.LogoURL = v
	}
	if v := over["logo_sha"]; v != "" {
		merged.LogoSHA = v
	}
	if v := over["primary_color"]; v != "" {
		merged.PrimaryColor = v
	}
	if v := over["secondary_color"]; v != "" {
		merged.SecondaryColor = v
	}
	if v := over["font_family"]; v != "" {
		merged.FontFamily = v
	}
	if v := over["footer_text"]; v != "" {
		merged.FooterText = v
	}
	if v := over["email_from_name"]; v != "" {
		merged.EmailFromName = v
	}
	if v := over["email_from_address"]; v != "" {
		merged.EmailFromAddress = v
	}
	return &merged, nil
}
