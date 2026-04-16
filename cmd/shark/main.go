package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/sharkauth/sharkauth/cmd/shark/cmd"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	cmd.SetMigrations(migrationsFS)
	if err := cmd.Execute(); err != nil {
		slog.Error("command failed", "error", err)
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
