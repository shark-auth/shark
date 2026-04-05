package storage

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// RunMigrations runs all pending goose migrations against the given database.
// The migrationsFS should contain SQL migration files embedded from the migrations/ directory.
// The dirPath is the path within the embed.FS where migrations are located.
func RunMigrations(db *sql.DB, migrationsFS embed.FS, dirPath string) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	if err := goose.Up(db, dirPath); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
