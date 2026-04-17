package storage

import (
	"context"
	"database/sql"
	"time"
)

// --- JWT signing keys ---

// InsertSigningKey inserts a new signing key row. Fails if an active key already
// exists (UNIQUE constraint on kid + DB-level CHECK on status).
func (s *SQLiteStore) InsertSigningKey(ctx context.Context, key *SigningKey) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jwt_signing_keys (kid, algorithm, public_key_pem, private_key_pem, status)
		 VALUES (?, ?, ?, ?, ?)`,
		key.KID, key.Algorithm, key.PublicKeyPEM, key.PrivateKeyPEM, key.Status,
	)
	return err
}

// GetActiveSigningKey returns the single active signing key, or sql.ErrNoRows.
func (s *SQLiteStore) GetActiveSigningKey(ctx context.Context) (*SigningKey, error) {
	return s.scanSigningKey(s.db.QueryRowContext(ctx,
		`SELECT id, kid, algorithm, public_key_pem, private_key_pem, created_at, rotated_at, status
		 FROM jwt_signing_keys WHERE status = 'active' LIMIT 1`))
}

// GetSigningKeyByKID returns a key by its kid (active or retired).
func (s *SQLiteStore) GetSigningKeyByKID(ctx context.Context, kid string) (*SigningKey, error) {
	return s.scanSigningKey(s.db.QueryRowContext(ctx,
		`SELECT id, kid, algorithm, public_key_pem, private_key_pem, created_at, rotated_at, status
		 FROM jwt_signing_keys WHERE kid = ?`, kid))
}

// RotateSigningKeys marks all active keys as retired (sets rotated_at=now), then
// inserts the new key as active. Both operations run in a transaction.
func (s *SQLiteStore) RotateSigningKeys(ctx context.Context, newKey *SigningKey) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx,
		`UPDATE jwt_signing_keys SET status = 'retired', rotated_at = ? WHERE status = 'active'`,
		now,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO jwt_signing_keys (kid, algorithm, public_key_pem, private_key_pem, status)
		 VALUES (?, ?, ?, ?, 'active')`,
		newKey.KID, newKey.Algorithm, newKey.PublicKeyPEM, newKey.PrivateKeyPEM,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// ListJWKSCandidates returns keys to include in a JWKS response.
//
//   - If activeOnly is true, only active keys are returned.
//   - If activeOnly is false, retired keys whose rotated_at is after retiredCutoff
//     are also included (keeps recently-rotated keys available for in-flight token
//     validation without a background cleanup job).
func (s *SQLiteStore) ListJWKSCandidates(ctx context.Context, activeOnly bool, retiredCutoff time.Time) ([]*SigningKey, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if activeOnly {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, kid, algorithm, public_key_pem, private_key_pem, created_at, rotated_at, status
			 FROM jwt_signing_keys WHERE status = 'active'
			 ORDER BY created_at DESC`)
	} else {
		cutoff := retiredCutoff.UTC().Format(time.RFC3339)
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, kid, algorithm, public_key_pem, private_key_pem, created_at, rotated_at, status
			 FROM jwt_signing_keys
			 WHERE status = 'active'
			    OR (status = 'retired' AND rotated_at > ?)
			 ORDER BY created_at DESC`, cutoff)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*SigningKey
	for rows.Next() {
		k, err := s.scanSigningKeyRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) scanSigningKey(row *sql.Row) (*SigningKey, error) {
	var k SigningKey
	var rotatedAt sql.NullString
	if err := row.Scan(&k.ID, &k.KID, &k.Algorithm, &k.PublicKeyPEM, &k.PrivateKeyPEM,
		&k.CreatedAt, &rotatedAt, &k.Status); err != nil {
		return nil, err
	}
	if rotatedAt.Valid {
		k.RotatedAt = &rotatedAt.String
	}
	return &k, nil
}

func (s *SQLiteStore) scanSigningKeyRow(rows *sql.Rows) (*SigningKey, error) {
	var k SigningKey
	var rotatedAt sql.NullString
	if err := rows.Scan(&k.ID, &k.KID, &k.Algorithm, &k.PublicKeyPEM, &k.PrivateKeyPEM,
		&k.CreatedAt, &rotatedAt, &k.Status); err != nil {
		return nil, err
	}
	if rotatedAt.Valid {
		k.RotatedAt = &rotatedAt.String
	}
	return &k, nil
}

// --- Revoked JTIs ---

// InsertRevokedJTI records a jti as revoked. Idempotent (INSERT OR IGNORE).
func (s *SQLiteStore) InsertRevokedJTI(ctx context.Context, jti string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO revoked_jti (jti, expires_at) VALUES (?, ?)`,
		jti, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

// IsRevokedJTI returns true if the jti exists in the revoked_jti table.
func (s *SQLiteStore) IsRevokedJTI(ctx context.Context, jti string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM revoked_jti WHERE jti = ?`, jti,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// PruneExpiredRevokedJTI deletes revoked_jti rows whose expires_at is in the past.
// This lazy cleanup prevents unbounded table growth without a background job.
func (s *SQLiteStore) PruneExpiredRevokedJTI(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM revoked_jti WHERE expires_at < datetime('now')`)
	return err
}
