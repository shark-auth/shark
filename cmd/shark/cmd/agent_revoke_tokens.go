package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var agentRevokeTokensCmd = &cobra.Command{
	Use:   "revoke-tokens <id>",
	Short: "Revoke all active tokens for an agent",
	Long:  `Revokes all active OAuth tokens for an agent via POST /api/v1/agents/{id}/tokens/revoke-all.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "POST", "/api/v1/agents/"+id+"/tokens/revoke-all", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "agent_not_found",
				fmt.Errorf("agent not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "revoke_failed",
				fmt.Errorf("revoke agent tokens: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		r := extractData(body)
		revoked, _ := r["revoked"].(float64)
		fmt.Fprintf(cmd.OutOrStdout(), "revoked %d tokens for agent %s\n", int(revoked), id)
		return nil
	},
}

func init() {
	addJSONFlag(agentRevokeTokensCmd)
	agentCmd.AddCommand(agentRevokeTokensCmd)
}
