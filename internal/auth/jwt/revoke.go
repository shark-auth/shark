package jwt

import (
	"context"
	"time"
)

// RevokeJTI marks a JTI as revoked in the store with lazy pruning.
// Prunes expired rows first (prevents unbounded growth).
func (m *Manager) RevokeJTI(ctx context.Context, jti string, expiresAt time.Time) error {
	// Lazy cleanup of expired revoked JTIs
	_ = m.store.PruneExpiredRevokedJTI(ctx)

	return m.store.InsertRevokedJTI(ctx, jti, expiresAt)
}
