package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var agentUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		payload := map[string]any{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			payload["name"] = v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			payload["description"] = v
		}

		if len(payload) == 0 {
			return fmt.Errorf("no fields to update — specify at least one flag")
		}

		body, code, err := adminDo(cmd, "PATCH", "/api/v1/agents/"+id, payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "agent_not_found",
				fmt.Errorf("agent not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "update_failed",
				fmt.Errorf("update agent: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "agent updated: %s\n", id)
		return nil
	},
}

func init() {
	agentUpdateCmd.Flags().String("name", "", "new agent name")
	agentUpdateCmd.Flags().String("description", "", "new description")
	addJSONFlag(agentUpdateCmd)
	agentCmd.AddCommand(agentUpdateCmd)
}
