// Package cmd wires cobra subcommands for the shark binary.
package cmd

import (
	"embed"

	"github.com/spf13/cobra"
)

// migrationsFS is injected by main.go at startup; holds the embedded migrations/.
var migrationsFS embed.FS

// SetMigrations is called once from main before Execute to inject the embed.FS.
func SetMigrations(fs embed.FS) { migrationsFS = fs }

// root is the base command for the shark binary.
var root = &cobra.Command{
	Use:   "shark",
	Short: "SharkAuth — single-binary identity platform",
	Long: `SharkAuth is a single Go binary that provides auth: password, OAuth,
passkeys, magic links, MFA, SSO, RBAC, organizations, audit logs,
agent auth — all embedded with SQLite.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return root.Execute()
}

func init() {
	root.AddCommand(serveCmd)
	root.AddCommand(initCmd)
	root.AddCommand(healthCmd)
	root.AddCommand(versionCmd)
}
