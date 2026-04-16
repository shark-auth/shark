package cmd

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/server"
)

var (
	serveConfigPath string
	serveDev        bool
	serveDevReset   bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the SharkAuth HTTP server",
	Long:  "Starts the HTTP server with the admin API, dashboard, and OAuth endpoints.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		opts := server.Options{
			ConfigPath:    serveConfigPath,
			MigrationsFS:  migrationsFS,
			MigrationsDir: "migrations",
		}
		if serveDev {
			if err := applyDevMode(&opts, serveDevReset); err != nil {
				return err
			}
		}

		return server.Serve(ctx, opts)
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveConfigPath, "config", "sharkauth.yaml", "path to config file")
	serveCmd.Flags().BoolVar(&serveDev, "dev", false, "enable dev mode (ephemeral storage, dev inbox, relaxed CORS)")
	serveCmd.Flags().BoolVar(&serveDevReset, "reset", false, "wipe dev.db before starting (requires --dev)")
}
