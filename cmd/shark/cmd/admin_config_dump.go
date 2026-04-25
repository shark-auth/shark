package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

// adminConfigCmd is the `shark admin config` subcommand group.
var adminConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect live server configuration",
}

var adminConfigDumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Print live config as JSON",
	Long: `Fetches and prints the running server's active configuration as JSON.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/config", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code != http.StatusOK {
			return maybeJSONErr(cmd, "config_dump_failed", fmt.Errorf("config dump: %s", apiError(body, code)))
		}
		return writeJSON(cmd.OutOrStdout(), body)
	},
}

func init() {
	adminConfigCmd.AddCommand(adminConfigDumpCmd)
	adminCmd.AddCommand(adminConfigCmd)
}
