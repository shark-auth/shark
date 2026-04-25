package storage_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

func TestSecrets_RoundTrip(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Absent key returns sql.ErrNoRows.
	_, err := store.GetSecret(ctx, "session_secret")
	if err != sql.ErrNoRows {
		t.Fatalf("missing key: want sql.ErrNoRows, got %v", err)
	}

	// Set then get.
	if err := store.SetSecret(ctx, "session_secret", "supersecret"); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}
	v, err := store.GetSecret(ctx, "session_secret")
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if v != "supersecret" {
		t.Errorf("want 'supersecret', got %q", v)
	}
}

func TestSecrets_UpdateRotatesTimestamp(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	if err := store.SetSecret(ctx, "k", "v1"); err != nil {
		t.Fatalf("SetSecret v1: %v", err)
	}
	if err := store.SetSecret(ctx, "k", "v2"); err != nil {
		t.Fatalf("SetSecret v2: %v", err)
	}

	v, err := store.GetSecret(ctx, "k")
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if v != "v2" {
		t.Errorf("want 'v2', got %q", v)
	}

	// Verify rotated_at is set via raw query.
	var rotatedAt *string
	if err := store.DB().QueryRowContext(ctx,
		`SELECT rotated_at FROM secrets WHERE name = 'k'`,
	).Scan(&rotatedAt); err != nil {
		t.Fatalf("scan rotated_at: %v", err)
	}
	if rotatedAt == nil {
		t.Error("rotated_at should be set after update, got NULL")
	}
}

func TestSecrets_Delete(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	if err := store.SetSecret(ctx, "tmp", "val"); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}
	if err := store.DeleteSecret(ctx, "tmp"); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}

	_, err := store.GetSecret(ctx, "tmp")
	if err != sql.ErrNoRows {
		t.Fatalf("after delete: want sql.ErrNoRows, got %v", err)
	}

	// Delete of absent key is no-op (no error).
	if err := store.DeleteSecret(ctx, "nonexistent"); err != nil {
		t.Fatalf("delete nonexistent: %v", err)
	}
}

func TestSecrets_IdempotentInsert(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// INSERT OR IGNORE via SetSecret called twice with same name must not fail.
	if err := store.SetSecret(ctx, "key", "a"); err != nil {
		t.Fatalf("first set: %v", err)
	}
	if err := store.SetSecret(ctx, "key", "b"); err != nil {
		t.Fatalf("second set: %v", err)
	}

	v, _ := store.GetSecret(ctx, "key")
	if v != "b" {
		t.Errorf("want 'b', got %q", v)
	}
}
