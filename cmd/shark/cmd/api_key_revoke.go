package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var apiKeyRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API key",
	Long:  `Revokes (deletes) an API key via DELETE /api/v1/api-keys/{id}.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "DELETE", "/api/v1/api-keys/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("API key not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "revoke_failed",
				fmt.Errorf("revoke API key: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{"revoked": id})
		}

		fmt.Fprintf(cmd.OutOrStdout(), "revoked API key %s\n", id)
		return nil
	},
}

func init() {
	addJSONFlag(apiKeyRevokeCmd)
	apiKeyCmd.AddCommand(apiKeyRevokeCmd)
}
