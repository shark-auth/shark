package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shark-auth/shark/internal/demo"
)

var (
	demoBaseURL  string
	demoAdminKey string
	demoHTMLOut  string
	demoOutput   string
	demoPlain    bool
	demoNoOpen   bool
	demoKeep     bool
	demoFast     bool
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run a SharkAuth demo flow",
}

var demoDelegationCmd = &cobra.Command{
	Use:   "delegation-with-trace",
	Short: "Run a 3-hop delegation chain demo with DPoP at each hop and vault retrieval",
	Long: `Registers a synthetic user + 3 agents (user-proxy, email-service, followup-service),
configures may_act policies, runs a 3-hop token-exchange chain with DPoP proofs at each hop,
fetches a vault token on hop 4, and generates a self-contained HTML report.
Requires a running shark server.

Examples:
  shark demo delegation-with-trace --admin-key <key>
  shark demo delegation-with-trace --admin-key <key> --output ./trace.html --no-open
  shark demo delegation-with-trace --admin-key <key> --keep`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outPath := demoOutput
		if outPath == "" {
			outPath = demoHTMLOut
		}
		if outPath == "" {
			outPath = "./demo-report.html"
		}

		opts := demo.DelegationOptions{
			BaseURL:  demoBaseURL,
			AdminKey: demoAdminKey,
			HTMLOut:  demoHTMLOut,
			Output:   outPath,
			Plain:    demoPlain,
			NoOpen:   demoNoOpen,
			Keep:     demoKeep,
			Fast:     demoFast,
		}
		if opts.AdminKey == "" {
			opts.AdminKey = os.Getenv("SHARK_ADMIN_KEY")
		}
		if opts.AdminKey == "" {
			return fmt.Errorf("admin key required: pass --admin-key or set SHARK_ADMIN_KEY")
		}
		return demo.RunDelegation(cmd.Context(), opts)
	},
}

func init() {
	demoDelegationCmd.Flags().StringVar(&demoBaseURL, "base-url", "http://localhost:8080", "shark server base URL")
	demoDelegationCmd.Flags().StringVar(&demoAdminKey, "admin-key", "", "admin API key (or SHARK_ADMIN_KEY env)")
	demoDelegationCmd.Flags().StringVar(&demoOutput, "output", "./demo-report.html", "write HTML report to this path")
	demoDelegationCmd.Flags().StringVar(&demoHTMLOut, "html", "", "alias for --output (deprecated)")
	demoDelegationCmd.Flags().BoolVar(&demoPlain, "plain", false, "force plain stdout output only (skip HTML)")
	demoDelegationCmd.Flags().BoolVar(&demoNoOpen, "no-open", false, "do not auto-open the report in a browser")
	demoDelegationCmd.Flags().BoolVar(&demoKeep, "keep", false, "keep temp DB for inspection after run")
	demoDelegationCmd.Flags().BoolVar(&demoFast, "fast", false, "skip screencast pacing delays (also: SHARK_DEMO_FAST=1)")
	demoCmd.AddCommand(demoDelegationCmd)
	root.AddCommand(demoCmd)
}
