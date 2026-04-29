// Package storage — user tier helpers (PROXYV1_5 §4.10).
//
// Tier (free / pro) is stored inside users.metadata JSON under the
// "tier" key rather than its own column. Rationale: metadata is already
// the bucket for app-defined per-user fields, and keeping tier there
// means the Lane A Claims baker + this helper share one source of truth
// without a schema migration.

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SetUserTier writes tier into users.metadata[tier]. Preserves every
// other key in metadata — callers commonly stash unrelated app data
// there. Returns sql.ErrNoRows when userID doesn't exist (so the admin
// handler can 404 cleanly).
func (s *SQLiteStore) SetUserTier(ctx context.Context, userID, tier string) error {
	var metaStr string
	row := s.reader.QueryRowContext(ctx, `SELECT metadata FROM users WHERE id = ?`, userID)
	if err := row.Scan(&metaStr); err != nil {
		return err
	}

	meta := map[string]any{}
	if metaStr != "" {
		if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
			// Corrupted JSON in the column: start fresh. A hard error here
			// would trap the admin out of ever recovering the row.
			meta = map[string]any{}
		}
	}
	meta["tier"] = tier

	out, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	res, err := s.writer.ExecContext(ctx,
		`UPDATE users SET metadata = ?, updated_at = ? WHERE id = ?`,
		string(out), time.Now().UTC().Format(time.RFC3339), userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetUserTier reads the "tier" key from users.metadata JSON. Returns ""
// (without error) when the user exists but has no tier set — callers
// default "" to "free" at the edge. Returns sql.ErrNoRows only when the
// user itself is missing.
func (s *SQLiteStore) GetUserTier(ctx context.Context, userID string) (string, error) {
	var metaStr string
	row := s.reader.QueryRowContext(ctx, `SELECT metadata FROM users WHERE id = ?`, userID)
	if err := row.Scan(&metaStr); err != nil {
		return "", err
	}
	if metaStr == "" {
		return "", nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
		// Corrupted JSON — treat as "no tier" rather than propagating.
		// Lane A's Claims baker follows the same policy.
		return "", nil
	}
	if v, ok := meta["tier"].(string); ok {
		return v, nil
	}
	return "", nil
}
