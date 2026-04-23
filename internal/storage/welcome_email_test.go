package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestMarkWelcomeEmailSent_Idempotent verifies the store-layer guarantee
// that MarkWelcomeEmailSent flips welcome_email_sent exactly once.
// First call succeeds; second call returns sql.ErrNoRows so callers can
// skip the welcome-email dispatch without an extra read.
func TestMarkWelcomeEmailSent_Idempotent(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Format(time.RFC3339)
	user := &storage.User{
		ID:        "usr_welcome_idem",
		Email:     "welcome-idem@example.com",
		HashType:  "argon2id",
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// First call — flag flips 0 -> 1, no error.
	if err := store.MarkWelcomeEmailSent(ctx, user.ID); err != nil {
		t.Fatalf("first MarkWelcomeEmailSent: expected nil, got %v", err)
	}

	// Second call — already 1, UPDATE matches zero rows, returns ErrNoRows.
	err := store.MarkWelcomeEmailSent(ctx, user.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second MarkWelcomeEmailSent: expected sql.ErrNoRows, got %v", err)
	}

	// Same behavior for a user that doesn't exist — no match, ErrNoRows.
	err = store.MarkWelcomeEmailSent(ctx, "usr_does_not_exist")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing user: expected sql.ErrNoRows, got %v", err)
	}
}
