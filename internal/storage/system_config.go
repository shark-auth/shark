package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SystemConfigRow is the raw DB row from the system_config table.
type SystemConfigRow struct {
	ID        int
	Payload   string
	UpdatedAt time.Time
}

// GetSystemConfig reads the single system_config row and returns the payload
// JSON string. Returns ("", nil) when the row exists but payload is empty.
// Returns ErrNotFound (sql.ErrNoRows) when the table has no row yet.
func (s *SQLiteStore) GetSystemConfig(ctx context.Context) (string, error) {
	var payload string
	err := s.db.QueryRowContext(ctx,
		`SELECT payload FROM system_config WHERE id = 1`,
	).Scan(&payload)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("storage: get system_config: %w", err)
	}
	return payload, nil
}

// SetSystemConfig JSON-marshals v and writes it into the single system_config
// row (upsert). The row is guaranteed to exist after the migration seed, but
// the upsert is kept for safety.
func (s *SQLiteStore) SetSystemConfig(ctx context.Context, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("storage: marshal system_config: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO system_config (id, payload, updated_at)
		 VALUES (1, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET
		   payload    = excluded.payload,
		   updated_at = excluded.updated_at`,
		string(b),
	)
	if err != nil {
		return fmt.Errorf("storage: set system_config: %w", err)
	}
	return nil
}
