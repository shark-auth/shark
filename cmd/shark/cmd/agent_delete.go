package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var agentDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Deactivate (delete) an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		yes, _ := cmd.Flags().GetBool("yes")

		if !yes {
			fmt.Printf("Deactivate agent %s? [y/N] ", id)
			r := bufio.NewReader(os.Stdin)
			line, _ := r.ReadString('\n')
			line = strings.TrimSpace(strings.ToLower(line))
			if line != "y" && line != "yes" {
				fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return nil
			}
		}

		body, code, err := adminDo(cmd, "DELETE", "/api/v1/agents/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "agent_not_found",
				fmt.Errorf("agent not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "delete_failed",
				fmt.Errorf("delete agent: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{"deleted": id})
		}

		fmt.Fprintf(cmd.OutOrStdout(), "agent deactivated: %s\n", id)
		return nil
	},
}

func init() {
	agentDeleteCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	addJSONFlag(agentDeleteCmd)
	agentCmd.AddCommand(agentDeleteCmd)
}
