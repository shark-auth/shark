package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var sessionRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke a session by ID",
	Long:  `Revokes a session via DELETE /api/v1/admin/sessions/{id}.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "DELETE", "/api/v1/admin/sessions/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("session not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "revoke_failed",
				fmt.Errorf("revoke session: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{"revoked": id})
		}

		fmt.Fprintf(cmd.OutOrStdout(), "revoked session %s\n", id)
		return nil
	},
}

func init() {
	addJSONFlag(sessionRevokeCmd)
	sessionCmd.AddCommand(sessionRevokeCmd)
}
