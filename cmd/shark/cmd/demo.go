package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/demo"
)

var (
	demoBaseURL  string
	demoAdminKey string
	demoHTMLOut  string
	demoPlain    bool
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run a SharkAuth demo flow",
}

var demoDelegationCmd = &cobra.Command{
	Use:   "delegation-with-trace",
	Short: "Run a 3-hop delegation chain demo with DPoP at each hop",
	Long: `Registers a synthetic user + 3 agents (user-proxy, email-service, followup-service),
configures may_act policies, runs a 3-hop token-exchange chain with DPoP proofs at each hop,
and reports the result. Requires a running shark server.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := demo.DelegationOptions{
			BaseURL:  demoBaseURL,
			AdminKey: demoAdminKey,
			HTMLOut:  demoHTMLOut,
			Plain:    demoPlain || demoHTMLOut == "",
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
	demoDelegationCmd.Flags().StringVar(&demoHTMLOut, "html", "", "(W+1) write HTML report to this path; empty = plain stdout")
	demoDelegationCmd.Flags().BoolVar(&demoPlain, "plain", true, "force plain stdout output")
	demoCmd.AddCommand(demoDelegationCmd)
	root.AddCommand(demoCmd)
}
