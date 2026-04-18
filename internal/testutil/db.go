package testutil

import (
	"embed"
	"testing"

	"github.com/sharkauth/sharkauth/internal/storage"
)

//go:embed migrations/*.sql
var testMigrationsFS embed.FS

// NewTestDB creates an in-memory SQLite database with all migrations applied.
// The database is automatically closed when the test completes.
func NewTestDB(t *testing.T) *storage.SQLiteStore {
	t.Helper()

	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("creating test db: %v", err)
	}

	if err := storage.RunMigrations(store.DB(), testMigrationsFS, "migrations"); err != nil {
		store.Close() //#nosec G104 -- cleanup after failed migration; test is already failing
		t.Fatalf("running migrations: %v", err)
	}

	t.Cleanup(func() {
		store.Close() //#nosec G104 -- test-cleanup close; nothing actionable on error
	})

	return store
}
