package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CreateProxyRule inserts a new override rule. Methods + Scopes are persisted
// as JSON arrays so an empty list round-trips to the same shape on read.
func (s *SQLiteStore) CreateProxyRule(ctx context.Context, rule *ProxyRule) error {
	methodsJSON, err := marshalStringSlice(rule.Methods)
	if err != nil {
		return fmt.Errorf("marshal methods: %w", err)
	}
	scopesJSON, err := marshalStringSlice(rule.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO proxy_rules
		 (id, name, pattern, methods, require, allow, scopes, enabled, priority, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.ID, rule.Name, rule.Pattern, methodsJSON, rule.Require, rule.Allow,
		scopesJSON, boolToInt(rule.Enabled), rule.Priority,
		rule.CreatedAt.UTC().Format(time.RFC3339),
		rule.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// GetProxyRuleByID returns a single rule or sql.ErrNoRows when missing.
func (s *SQLiteStore) GetProxyRuleByID(ctx context.Context, id string) (*ProxyRule, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, pattern, methods, require, allow, scopes, enabled, priority, created_at, updated_at
		 FROM proxy_rules WHERE id = ?`, id)
	return scanProxyRuleRow(row)
}

// ListProxyRules returns rules ordered priority DESC, then created_at ASC so
// ties are stable across reloads. Disabled rules are included — the engine
// loader is responsible for filtering them out.
func (s *SQLiteStore) ListProxyRules(ctx context.Context) ([]*ProxyRule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, pattern, methods, require, allow, scopes, enabled, priority, created_at, updated_at
		 FROM proxy_rules ORDER BY priority DESC, created_at ASC`)
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

// UpdateProxyRule persists a partial-or-full update. Caller is expected to
// have already loaded + mutated the row so we can write every column without
// branching — keeps the SQL identical to the create path's column list.
func (s *SQLiteStore) UpdateProxyRule(ctx context.Context, rule *ProxyRule) error {
	methodsJSON, err := marshalStringSlice(rule.Methods)
	if err != nil {
		return fmt.Errorf("marshal methods: %w", err)
	}
	scopesJSON, err := marshalStringSlice(rule.Scopes)
	if err != nil {
		return fmt.Errorf("marshal scopes: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE proxy_rules SET
		   name = ?, pattern = ?, methods = ?, require = ?, allow = ?,
		   scopes = ?, enabled = ?, priority = ?, updated_at = ?
		 WHERE id = ?`,
		rule.Name, rule.Pattern, methodsJSON, rule.Require, rule.Allow,
		scopesJSON, boolToInt(rule.Enabled), rule.Priority,
		time.Now().UTC().Format(time.RFC3339),
		rule.ID,
	)
	return err
}

// DeleteProxyRule is a no-op when the id doesn't exist; callers that need to
// distinguish 404 vs 204 should GetProxyRuleByID first.
func (s *SQLiteStore) DeleteProxyRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM proxy_rules WHERE id = ?`, id)
	return err
}

// scanProxyRuleRow handles QueryRowContext output.
func scanProxyRuleRow(row *sql.Row) (*ProxyRule, error) {
	var r ProxyRule
	var methodsJSON, scopesJSON string
	var enabled int
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&r.ID, &r.Name, &r.Pattern, &methodsJSON, &r.Require, &r.Allow,
		&scopesJSON, &enabled, &r.Priority,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return finalizeProxyRule(&r, methodsJSON, scopesJSON, enabled, createdAtStr, updatedAtStr)
}

// scanProxyRuleRows handles QueryContext (rows iterator) output.
func scanProxyRuleRows(rows *sql.Rows) (*ProxyRule, error) {
	var r ProxyRule
	var methodsJSON, scopesJSON string
	var enabled int
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&r.ID, &r.Name, &r.Pattern, &methodsJSON, &r.Require, &r.Allow,
		&scopesJSON, &enabled, &r.Priority,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return finalizeProxyRule(&r, methodsJSON, scopesJSON, enabled, createdAtStr, updatedAtStr)
}

// finalizeProxyRule decodes JSON columns + integer/timestamp normalisation.
// Empty Methods/Scopes are returned as []string{} (never nil) so the wire
// layer never has to fold a nil case.
func finalizeProxyRule(
	r *ProxyRule,
	methodsJSON, scopesJSON string,
	enabled int,
	createdAtStr, updatedAtStr string,
) (*ProxyRule, error) {
	r.Enabled = enabled != 0

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

// unmarshalStringSlice decodes a JSON column into a []string, normalising
// empty/nil inputs to []string{} (non-nil) so callers never deal with nil.
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

// parseProxyRuleTime accepts both RFC3339 (what we write) and SQLite's default
// CURRENT_TIMESTAMP format ("2006-01-02 15:04:05"). Mirrors parseAuthFlowTime.
func parseProxyRuleTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}
