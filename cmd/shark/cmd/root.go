// Package cmd wires cobra subcommands for the shark binary.
package cmd

import (
	"embed"
	"encoding/json"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/cli"
)

// migrationsFS is injected by main.go at startup; holds the embedded migrations/.
var migrationsFS embed.FS

// SetMigrations is called once from main before Execute to inject the embed.FS.
func SetMigrations(fs embed.FS) { migrationsFS = fs }

// verbose is wired to the root command's persistent --verbose/-v flag.
// When true, the default slog logger is upgraded to DEBUG level writing to stderr.
var verbose bool

// root is the base command for the shark binary.
var root = &cobra.Command{
	Use:   "shark",
	Short: "SharkAuth — single-binary identity platform",
	Long: `SharkAuth is a single Go binary that provides auth: password, OAuth,
passkeys, magic links, MFA, SSO, RBAC, organizations, audit logs,
agent auth — all embedded with SQLite.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configureLogger(verbose)
	},
}

// Execute runs the root command.
func Execute() error {
	return root.Execute()
}

// configureLogger wires slog.Default() to stderr at INFO or DEBUG depending on verbose.
// When stderr is a TTY (and NO_COLOR is not set), uses the pretty PrettyHandler;
// otherwise falls back to the standard text handler for log-aggregator parsability.
func configureLogger(v bool) {
	level := slog.LevelInfo
	if v {
		level = slog.LevelDebug
	}
	slog.SetDefault(cli.NewServerLogger(os.Stderr, level))
}

// jsonFlag returns true if the command has a --json flag and it is set.
func jsonFlag(cmd *cobra.Command) bool {
	if f := cmd.Flags().Lookup("json"); f != nil {
		v, _ := cmd.Flags().GetBool("json")
		return v
	}
	return false
}

// writeJSON encodes payload as indented JSON to w.
func writeJSON(w io.Writer, payload any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// writeJSONError writes a structured JSON error payload to stderr and returns err unchanged.
// Callers should: if jsonFlag(cmd) { return writeJSONError(cmd, "code", err, details) } else { return err }.
func writeJSONError(cmd *cobra.Command, code string, err error, details map[string]any) error {
	payload := map[string]any{
		"error":   code,
		"message": err.Error(),
	}
	if details != nil {
		payload["details"] = details
	}
	enc := json.NewEncoder(cmd.ErrOrStderr())
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
	return err
}

// addJSONFlag registers a local --json bool flag on cmd.
func addJSONFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "emit machine-readable JSON to stdout")
}

func init() {
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug-level logging to stderr")
	root.AddCommand(serveCmd)
	root.AddCommand(healthCmd)
	root.AddCommand(versionCmd)
	root.AddCommand(keysCmd)
	root.AddCommand(doctorCmd)
}
