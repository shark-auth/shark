package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/shark-auth/shark/internal/cli"
	"github.com/shark-auth/shark/internal/server"
)

var (
	serveProxyUpstream string
	serveNoPrompt      bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the SharkAuth HTTP server",
	Long:  "Starts the HTTP server with the admin API, dashboard, and OAuth endpoints.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// W18: print branded header on every boot so the operator has a
		// visible signal the binary started, plus version + size + URLs.
		cli.PrintHeader(os.Stdout)

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		opts := server.Options{
			MigrationsFS:  migrationsFS,
			MigrationsDir: "migrations",
			NoPrompt:      serveNoPrompt,
		}
		if serveProxyUpstream != "" {
			opts.ProxyUpstream = serveProxyUpstream
		}

		return server.Serve(ctx, opts)
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveProxyUpstream, "proxy-upstream", "", "mount reverse proxy to this upstream URL (e.g. http://localhost:3000)")
	serveCmd.Flags().BoolVar(&serveNoPrompt, "no-prompt", false, "skip first-boot browser-open prompt (for CI / headless environments)")
}
