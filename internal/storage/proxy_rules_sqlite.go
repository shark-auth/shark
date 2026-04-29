package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CreateProxyRule inserts a new override rule. Writes tier_match + m2m
// alongside the legacy columns so every Lane A/B enhancement round-trips
// via the same path dashboards + YAML import use.
func (s *SQLiteStore) CreateProxyRule(ctx context.Context, rule *ProxyRule) error {
	methodsJSON, err := marshalStringSlice(rule.Methods)
	if err != nil {
		return fmt.Errorf("marshal methods: %w", err)
	}
	scopesJSON, err := marshalStringSlice(rule.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}

	_, err = s.writer.ExecContext(ctx,
		`INSERT INTO proxy_rules
		 (id, app_id, name, pattern, methods, require, allow, scopes, enabled, priority, tier_match, m2m, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.AppID, rule.Name, rule.Pattern, methodsJSON, rule.Require, rule.Allow,
		scopesJSON, boolToInt(rule.Enabled), rule.Priority,
		rule.TierMatch, boolToInt(rule.M2M),
		rule.CreatedAt.UTC().Format(time.RFC3339),
		rule.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// proxyRuleSelectCols is the canonical SELECT column list. Kept as a
// package-level constant so every read path (single-row, multi-row,
// app-scoped) scans the same ordered columns — if columns ever get added
// or reordered there's one place to change.
const proxyRuleSelectCols = `id, app_id, name, pattern, methods, require, allow, scopes, enabled, priority, tier_match, m2m, created_at, updated_at`

// GetProxyRuleByID returns a single rule or sql.ErrNoRows when missing.
func (s *SQLiteStore) GetProxyRuleByID(ctx context.Context, id string) (*ProxyRule, error) {
	row := s.reader.QueryRowContext(ctx,
		`SELECT `+proxyRuleSelectCols+` FROM proxy_rules WHERE id = ?`, id)
	return scanProxyRuleRow(row)
}

// ListProxyRules returns all rules.
func (s *SQLiteStore) ListProxyRules(ctx context.Context) ([]*ProxyRule, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT `+proxyRuleSelectCols+` FROM proxy_rules ORDER BY priority DESC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ProxyRule
	for rows.Next() {
		r, err := scanProxyRuleRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListProxyRulesByAppID returns rules for a specific application.
func (s *SQLiteStore) ListProxyRulesByAppID(ctx context.Context, appID string) ([]*ProxyRule, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT `+proxyRuleSelectCols+` FROM proxy_rules WHERE app_id = ? ORDER BY priority DESC, created_at ASC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ProxyRule
	for rows.Next() {
		r, err := scanProxyRuleRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateProxyRule persists a partial-or-full update.
func (s *SQLiteStore) UpdateProxyRule(ctx context.Context, rule *ProxyRule) error {
	methodsJSON, err := marshalStringSlice(rule.Methods)
	if err != nil {
		return fmt.Errorf("marshal methods: %w", err)
	}
	scopesJSON, err := marshalStringSlice(rule.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}

	_, err = s.writer.ExecContext(ctx,
		`UPDATE proxy_rules SET
		   app_id = ?, name = ?, pattern = ?, methods = ?, require = ?, allow = ?,
		   scopes = ?, enabled = ?, priority = ?, tier_match = ?, m2m = ?, updated_at = ?
		 WHERE id = ?`,
		rule.AppID, rule.Name, rule.Pattern, methodsJSON, rule.Require, rule.Allow,
		scopesJSON, boolToInt(rule.Enabled), rule.Priority,
		rule.TierMatch, boolToInt(rule.M2M),
		time.Now().UTC().Format(time.RFC3339),
		rule.ID,
	)
	return err
}

// DeleteProxyRule deletes a rule by ID.
func (s *SQLiteStore) DeleteProxyRule(ctx context.Context, id string) error {
	_, err := s.writer.ExecContext(ctx, `DELETE FROM proxy_rules WHERE id = ?`, id)
	return err
}

// scanProxyRuleRow handles QueryRowContext output. Column order MUST
// mirror proxyRuleSelectCols.
func scanProxyRuleRow(row *sql.Row) (*ProxyRule, error) {
	var r ProxyRule
	var methodsJSON, scopesJSON string
	var enabled, m2m int
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&r.ID, &r.AppID, &r.Name, &r.Pattern, &methodsJSON, &r.Require, &r.Allow,
		&scopesJSON, &enabled, &r.Priority, &r.TierMatch, &m2m,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return finalizeProxyRule(&r, methodsJSON, scopesJSON, enabled, m2m, createdAtStr, updatedAtStr)
}

// scanProxyRuleRows handles QueryContext (rows iterator) output. Column
// order MUST mirror proxyRuleSelectCols.
func scanProxyRuleRows(rows *sql.Rows) (*ProxyRule, error) {
	var r ProxyRule
	var methodsJSON, scopesJSON string
	var enabled, m2m int
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&r.ID, &r.AppID, &r.Name, &r.Pattern, &methodsJSON, &r.Require, &r.Allow,
		&scopesJSON, &enabled, &r.Priority, &r.TierMatch, &m2m,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return finalizeProxyRule(&r, methodsJSON, scopesJSON, enabled, m2m, createdAtStr, updatedAtStr)
}

// finalizeProxyRule decodes JSON columns + integer/timestamp normalisation.
// Shared between the single-row and multi-row scan paths so the integer →
// bool conversion for Enabled + M2M lives in one place.
func finalizeProxyRule(
	r *ProxyRule,
	methodsJSON, scopesJSON string,
	enabled, m2m int,
	createdAtStr, updatedAtStr string,
) (*ProxyRule, error) {
	r.Enabled = enabled != 0
	r.M2M = m2m != 0

	methods, err := unmarshalStringSlice(methodsJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal methods: %w", err)
	}
	r.Methods = methods

	scopes, err := unmarshalStringSlice(scopesJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal scopes: %w", err)
	}
	r.Scopes = scopes

	r.CreatedAt, err = parseProxyRuleTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at %q: %w", createdAtStr, err)
	}
	r.UpdatedAt, err = parseProxyRuleTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at %q: %w", updatedAtStr, err)
	}
	return r, nil
}

// unmarshalStringSlice decodes a JSON column into a []string.
func unmarshalStringSlice(s string) ([]string, error) {
	if s == "" {
		return []string{}, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// parseProxyRuleTime accepts both RFC3339 and current timestamp format.
func parseProxyRuleTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}
