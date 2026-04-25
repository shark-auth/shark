package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// adminConfigCmd groups `shark admin config` subcommands.
var adminConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Admin configuration subcommands",
}

var adminConfigDumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump the current admin configuration",
	Long:  `Fetches and pretty-prints the live admin configuration via GET /api/v1/admin/config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/config", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "dump_failed",
				fmt.Errorf("dump config: %s", apiError(body, code)))
		}

		// Always emit JSON for config dump — it's inherently structured.
		return writeJSON(cmd.OutOrStdout(), body)
	},
}

func init() {
	addJSONFlag(adminConfigDumpCmd)
	adminConfigCmd.AddCommand(adminConfigDumpCmd)
	adminCmd.AddCommand(adminConfigCmd)
}
