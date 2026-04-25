package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var ssoUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an SSO connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		payload := map[string]any{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			payload["name"] = v
		}
		if cmd.Flags().Changed("domain") {
			v, _ := cmd.Flags().GetString("domain")
			payload["domain"] = v
		}
		if cmd.Flags().Changed("enabled") {
			v, _ := cmd.Flags().GetBool("enabled")
			payload["enabled"] = v
		}

		if len(payload) == 0 {
			return fmt.Errorf("no fields to update — specify at least one flag")
		}

		body, code, err := adminDo(cmd, "PUT", "/api/v1/sso/connections/"+id, payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("SSO connection not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "update_failed",
				fmt.Errorf("update SSO connection: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "SSO connection updated: %s\n", id)
		return nil
	},
}

func init() {
	ssoUpdateCmd.Flags().String("name", "", "new connection name")
	ssoUpdateCmd.Flags().String("domain", "", "new email domain")
	ssoUpdateCmd.Flags().Bool("enabled", true, "enable or disable the connection")
	addJSONFlag(ssoUpdateCmd)
	ssoCmd.AddCommand(ssoUpdateCmd)
}
