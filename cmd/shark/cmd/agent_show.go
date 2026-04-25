package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var agentShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "GET", "/api/v1/agents/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "agent_not_found",
				fmt.Errorf("agent not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show agent: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		a := extractData(body)
		aid, _ := a["id"].(string)
		name, _ := a["name"].(string)
		clientID, _ := a["client_id"].(string)
		active, _ := a["active"].(bool)
		created, _ := a["created_at"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "  id:         %s\n", aid)
		fmt.Fprintf(cmd.OutOrStdout(), "  name:       %s\n", name)
		fmt.Fprintf(cmd.OutOrStdout(), "  client_id:  %s\n", clientID)
		fmt.Fprintf(cmd.OutOrStdout(), "  active:     %v\n", active)
		fmt.Fprintf(cmd.OutOrStdout(), "  created_at: %s\n", created)
		return nil
	},
}

func init() {
	addJSONFlag(agentShowCmd)
	agentCmd.AddCommand(agentShowCmd)
}
