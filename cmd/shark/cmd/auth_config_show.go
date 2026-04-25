package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authConfigShowCmd = &cobra.Command{
	Use:   "config show",
	Short: "Show the current auth configuration",
	Long:  `Fetches the current auth configuration via GET /api/v1/admin/config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/config", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show auth config: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		// Pretty-print selected top-level fields
		cfg := extractData(body)
		if cfg == nil {
			return writeJSON(cmd.OutOrStdout(), body)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "auth configuration:")
		for k, v := range cfg {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s: %v\n", k, v)
		}
		return nil
	},
}

// configShowCmd is also registered as `shark auth config show`.
var configShowCmdAlias = &cobra.Command{
	Use:   "show",
	Short: "Show the current auth configuration",
	Long:  `Fetches the current auth configuration via GET /api/v1/admin/config.`,
	RunE:  authConfigShowCmd.RunE,
}

func init() {
	addJSONFlag(authConfigShowCmd)
	addJSONFlag(configShowCmdAlias)

	// Register as `shark auth config show`
	authConfigSubCmd := &cobra.Command{
		Use:   "config",
		Short: "Auth configuration subcommands",
	}
	authConfigSubCmd.AddCommand(configShowCmdAlias)
	authCmd.AddCommand(authConfigSubCmd)
}
