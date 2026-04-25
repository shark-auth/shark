package server

import (
	"context"
	"testing"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestFirstBootGeneratesSecrets verifies that RunFirstBoot on a fresh DB
// generates all three secrets (server.secret, admin.api_key.hash,
// jwt signing key via GetActiveSigningKey) and returns a non-nil result
// containing an AdminKey.
func TestFirstBootGeneratesSecrets(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	cfg := testutil.TestConfig()

	opts := Options{
		NoPrompt: true, // suppress interactive TTY prompts in CI
	}

	result, err := RunFirstBoot(ctx, store, cfg, opts)
	if err != nil {
		t.Fatalf("RunFirstBoot: %v", err)
	}
	if result == nil {
		t.Fatal("RunFirstBoot returned nil result on a fresh DB — expected first-boot secrets")
	}
	if result.AdminKey == "" {
		t.Error("AdminKey is empty after first boot")
	}
	if result.JWTKid == "" {
		t.Error("JWTKid is empty after first boot")
	}

	// Verify server.secret was stored in DB.
	serverSecret, err := store.GetSecret(ctx, "server.secret")
	if err != nil {
		t.Fatalf("server.secret not stored: %v", err)
	}
	if len(serverSecret) < 32 {
		t.Errorf("server.secret too short: %q", serverSecret)
	}

	// Verify admin.api_key.hash was stored.
	keyHash, err := store.GetSecret(ctx, "admin.api_key.hash")
	if err != nil {
		t.Fatalf("admin.api_key.hash not stored: %v", err)
	}
	if keyHash == "" {
		t.Error("admin.api_key.hash is empty")
	}

	// Verify JWT signing key was generated.
	activeKey, err := store.GetActiveSigningKey(ctx)
	if err != nil {
		t.Fatalf("GetActiveSigningKey: %v", err)
	}
	if activeKey.KID == "" {
		t.Error("active signing key has empty KID after first boot")
	}
}

// TestFirstBootIdempotent verifies that a second call to RunFirstBoot
// on the same DB is a no-op (returns nil, nil) and does NOT overwrite
// the secrets generated on the first call.
func TestFirstBootIdempotent(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	cfg := testutil.TestConfig()
	opts := Options{NoPrompt: true}

	// First call — should succeed.
	result1, err := RunFirstBoot(ctx, store, cfg, opts)
	if err != nil {
		t.Fatalf("first RunFirstBoot: %v", err)
	}
	if result1 == nil {
		t.Fatal("first RunFirstBoot returned nil on fresh DB")
	}

	// Capture the secret stored after the first boot.
	secretAfterFirst, err := store.GetSecret(ctx, "server.secret")
	if err != nil {
		t.Fatalf("read server.secret after first boot: %v", err)
	}

	// Second call — must be a no-op.
	result2, err := RunFirstBoot(ctx, store, cfg, opts)
	if err != nil {
		t.Fatalf("second RunFirstBoot: %v", err)
	}
	if result2 != nil {
		t.Error("second RunFirstBoot should return nil (no-op), got non-nil result")
	}

	// Confirm the secret was NOT replaced.
	secretAfterSecond, err := store.GetSecret(ctx, "server.secret")
	if err != nil {
		t.Fatalf("read server.secret after second boot: %v", err)
	}
	if secretAfterFirst != secretAfterSecond {
		t.Error("server.secret changed between first and second boot — idempotency broken")
	}
}

// TestFirstBootNonTTY verifies that RunFirstBoot with NoPrompt=true
// (the non-TTY / CI path) completes without error and still generates
// all secrets on a fresh DB.
func TestFirstBootNonTTY(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Build a minimal config — similar to what a non-interactive operator would have.
	cfg := &config.Config{}
	cfg.Server.Port = 8080
	cfg.Storage.Path = ":memory:"

	opts := Options{NoPrompt: true}

	result, err := RunFirstBoot(ctx, store, cfg, opts)
	if err != nil {
		t.Fatalf("RunFirstBoot (non-TTY): %v", err)
	}
	if result == nil {
		t.Fatal("RunFirstBoot (non-TTY) returned nil on fresh DB")
	}
	if result.AdminKey == "" {
		t.Error("AdminKey empty in non-TTY first boot")
	}

	// Ensure a secret was stored.
	if _, err := store.GetSecret(ctx, "server.secret"); err != nil {
		t.Fatalf("server.secret not stored in non-TTY boot: %v", err)
	}
}
