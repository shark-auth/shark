package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/server"
)

var (
	serveConfigPath    string
	serveDev           bool
	serveDevReset      bool
	serveProxyUpstream string
	serveNoPrompt      bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the SharkAuth HTTP server",
	Long:  "Starts the HTTP server with the admin API, dashboard, and OAuth endpoints.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		configPath := serveConfigPath
		if serveDev {
			// In dev mode a missing config is expected — don't force the user
			// to run `shark init` first. Only load the YAML if it exists.
			if _, err := os.Stat(configPath); err != nil {
				configPath = ""
			}
		}

		opts := server.Options{
			ConfigPath:    configPath,
			MigrationsFS:  migrationsFS,
			MigrationsDir: "migrations",
			NoPrompt:      serveNoPrompt,
		}
		if serveDev {
			if err := applyDevMode(&opts, serveDevReset); err != nil {
				return err
			}
		}
		if serveProxyUpstream != "" {
			opts.ProxyUpstream = serveProxyUpstream
		}

		return server.Serve(ctx, opts)
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveConfigPath, "config", "sharkauth.yaml", "path to config file")
	serveCmd.Flags().BoolVar(&serveDev, "dev", false, "enable dev mode (ephemeral storage, dev inbox, relaxed CORS)")
	serveCmd.Flags().BoolVar(&serveDevReset, "reset", false, "wipe dev.db before starting (requires --dev)")
	serveCmd.Flags().StringVar(&serveProxyUpstream, "proxy-upstream", "", "mount reverse proxy to this upstream URL (e.g. http://localhost:3000)")
	serveCmd.Flags().BoolVar(&serveNoPrompt, "no-prompt", false, "skip first-boot browser-open prompt (for CI / headless environments)")
}
