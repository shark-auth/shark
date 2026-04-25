package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var agentRotateSecretCmd = &cobra.Command{
	Use:   "rotate-secret <id>",
	Short: "Rotate an agent's client secret",
	Long:  `Rotates the agent's client secret via POST /api/v1/agents/{id}/rotate-secret. The new secret is shown only once.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "POST", "/api/v1/agents/"+id+"/rotate-secret", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "agent_not_found",
				fmt.Errorf("agent not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "rotate_failed",
				fmt.Errorf("rotate agent secret: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		a := extractData(body)
		secret, _ := a["client_secret"].(string)
		fmt.Fprintf(cmd.OutOrStdout(), "agent secret rotated: %s\n", id)
		if secret != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  client_secret: %s  (shown once — save it now)\n", secret)
		}
		return nil
	},
}

func init() {
	addJSONFlag(agentRotateSecretCmd)
	agentCmd.AddCommand(agentRotateSecretCmd)
}
