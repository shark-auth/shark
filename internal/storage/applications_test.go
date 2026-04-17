package storage_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

func newTestApp(t *testing.T, name string, isDefault bool) *storage.Application {
	t.Helper()
	id, err := gonanoid.New(21)
	if err != nil {
		t.Fatalf("generate nanoid: %v", err)
	}
	clientID, err := gonanoid.New(21)
	if err != nil {
		t.Fatalf("generate nanoid: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return &storage.Application{
		ID:                  "app_" + id,
		Name:                name,
		ClientID:            "shark_app_" + clientID,
		ClientSecretHash:    "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		ClientSecretPrefix:  "deadbeef",
		AllowedCallbackURLs: []string{"https://example.com/callback"},
		AllowedLogoutURLs:   []string{"https://example.com/logout"},
		AllowedOrigins:      []string{"https://example.com"},
		IsDefault:           isDefault,
		Metadata:            map[string]any{"env": "test"},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func TestApplicationCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	app := newTestApp(t, "My App", false)

	// Create
	if err := store.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}

	// GetByID
	got, err := store.GetApplicationByID(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApplicationByID: %v", err)
	}
	if got.Name != app.Name {
		t.Errorf("Name: got %q, want %q", got.Name, app.Name)
	}
	if got.ClientID != app.ClientID {
		t.Errorf("ClientID: got %q, want %q", got.ClientID, app.ClientID)
	}
	if got.ClientSecretHash != app.ClientSecretHash {
		t.Errorf("ClientSecretHash mismatch")
	}
	if len(got.AllowedCallbackURLs) != 1 || got.AllowedCallbackURLs[0] != "https://example.com/callback" {
		t.Errorf("AllowedCallbackURLs: got %v", got.AllowedCallbackURLs)
	}
	if got.IsDefault != false {
		t.Errorf("IsDefault: got %v, want false", got.IsDefault)
	}
	if got.Metadata["env"] != "test" {
		t.Errorf("Metadata[env]: got %v, want test", got.Metadata["env"])
	}

	// GetByClientID
	got2, err := store.GetApplicationByClientID(ctx, app.ClientID)
	if err != nil {
		t.Fatalf("GetApplicationByClientID: %v", err)
	}
	if got2.ID != app.ID {
		t.Errorf("ID via ClientID lookup: got %q, want %q", got2.ID, app.ID)
	}

	// List
	apps, err := store.ListApplications(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListApplications: %v", err)
	}
	if len(apps) != 1 {
		t.Fatalf("ListApplications: got %d, want 1", len(apps))
	}

	// Update
	app.Name = "Renamed App"
	app.AllowedCallbackURLs = []string{"https://example.com/cb2"}
	app.Metadata = map[string]any{"env": "prod"}
	if err := store.UpdateApplication(ctx, app); err != nil {
		t.Fatalf("UpdateApplication: %v", err)
	}
	updated, err := store.GetApplicationByID(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApplicationByID after update: %v", err)
	}
	if updated.Name != "Renamed App" {
		t.Errorf("Name after update: got %q, want %q", updated.Name, "Renamed App")
	}
	if len(updated.AllowedCallbackURLs) != 1 || updated.AllowedCallbackURLs[0] != "https://example.com/cb2" {
		t.Errorf("AllowedCallbackURLs after update: %v", updated.AllowedCallbackURLs)
	}
	if updated.Metadata["env"] != "prod" {
		t.Errorf("Metadata[env] after update: got %v, want prod", updated.Metadata["env"])
	}

	// RotateApplicationSecret
	newHash := "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
	newPrefix := "aabbccdd"
	if err := store.RotateApplicationSecret(ctx, app.ID, newHash, newPrefix); err != nil {
		t.Fatalf("RotateApplicationSecret: %v", err)
	}
	rotated, err := store.GetApplicationByID(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApplicationByID after rotate: %v", err)
	}
	if rotated.ClientSecretHash != newHash {
		t.Errorf("ClientSecretHash after rotate: got %q, want %q", rotated.ClientSecretHash, newHash)
	}
	if rotated.ClientSecretPrefix != newPrefix {
		t.Errorf("ClientSecretPrefix after rotate: got %q, want %q", rotated.ClientSecretPrefix, newPrefix)
	}

	// Delete
	if err := store.DeleteApplication(ctx, app.ID); err != nil {
		t.Fatalf("DeleteApplication: %v", err)
	}
	_, err = store.GetApplicationByID(ctx, app.ID)
	if err != sql.ErrNoRows {
		t.Errorf("GetApplicationByID after delete: got %v, want sql.ErrNoRows", err)
	}
}

func TestApplicationDefaultUniqueIndex(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Insert the first default application — must succeed.
	first := newTestApp(t, "Default App", true)
	if err := store.CreateApplication(ctx, first); err != nil {
		t.Fatalf("CreateApplication (first default): %v", err)
	}

	// GetDefaultApplication must return the first one.
	def, err := store.GetDefaultApplication(ctx)
	if err != nil {
		t.Fatalf("GetDefaultApplication: %v", err)
	}
	if def.ID != first.ID {
		t.Errorf("GetDefaultApplication ID: got %q, want %q", def.ID, first.ID)
	}

	// Insert a second application also marked is_default=1 — must fail due to partial unique index.
	second := newTestApp(t, "Second Default App", true)
	err = store.CreateApplication(ctx, second)
	if err == nil {
		t.Fatal("CreateApplication (second default): expected error due to unique index, got nil")
	}
}

func TestApplicationEmptyURLSlices(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Create an app with empty slices — they must round-trip as empty (not nil) slices.
	id, _ := gonanoid.New(21)
	cid, _ := gonanoid.New(21)
	now := time.Now().UTC().Truncate(time.Second)
	app := &storage.Application{
		ID:                  "app_" + id,
		Name:                "Empty Slices App",
		ClientID:            "shark_app_" + cid,
		ClientSecretHash:    "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		ClientSecretPrefix:  "deadbeef",
		AllowedCallbackURLs: nil, // nil → should be stored as "[]"
		AllowedLogoutURLs:   []string{},
		AllowedOrigins:      []string{},
		IsDefault:           false,
		Metadata:            nil, // nil → should be stored as "{}"
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := store.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication with empty slices: %v", err)
	}
	got, err := store.GetApplicationByID(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApplicationByID: %v", err)
	}
	if got.AllowedCallbackURLs == nil {
		t.Error("AllowedCallbackURLs should be non-nil empty slice, got nil")
	}
	if len(got.AllowedCallbackURLs) != 0 {
		t.Errorf("AllowedCallbackURLs should be empty, got %v", got.AllowedCallbackURLs)
	}
	if got.Metadata == nil {
		t.Error("Metadata should be non-nil empty map, got nil")
	}
}
