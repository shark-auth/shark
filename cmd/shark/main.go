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
		// Allow subcommands (e.g. shark doctor) to request a specific exit code
		// by wrapping an ExitCoder in their returned error.
		if ec, ok := cmd.AsExitCoder(err); ok {
			os.Exit(ec)
		}
		slog.Error("command failed", "error", err)
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
