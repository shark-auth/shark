package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Secret is a named secret stored in the secrets table.
type Secret struct {
	Name      string
	Value     string
	CreatedAt time.Time
	RotatedAt *time.Time
}

// GetSecret retrieves a named secret. Returns ("", sql.ErrNoRows) when absent.
func (s *SQLiteStore) GetSecret(ctx context.Context, name string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx,
		`SELECT value FROM secrets WHERE name = ?`, name,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", sql.ErrNoRows
	}
	if err != nil {
		return "", fmt.Errorf("storage: get secret %q: %w", name, err)
	}
	return value, nil
}

// SetSecret inserts or replaces a named secret, updating rotated_at when the
// row already exists.
func (s *SQLiteStore) SetSecret(ctx context.Context, name, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO secrets (name, value, created_at)
		 VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(name) DO UPDATE SET
		   value      = excluded.value,
		   rotated_at = CURRENT_TIMESTAMP`,
		name, value,
	)
	if err != nil {
		return fmt.Errorf("storage: set secret %q: %w", name, err)
	}
	return nil
}

// DeleteSecret removes a named secret. No-op when absent.
func (s *SQLiteStore) DeleteSecret(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM secrets WHERE name = ?`, name,
	)
	if err != nil {
		return fmt.Errorf("storage: delete secret %q: %w", name, err)
	}
	return nil
}
