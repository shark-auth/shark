package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var vaultProviderShowCmd = &cobra.Command{
	Use:   "show <name-or-id>",
	Short: "Show a vault provider",
	Long:  `Show a vault provider by ID via GET /api/v1/vault/providers/{id}.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "GET", "/api/v1/vault/providers/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("vault provider not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show vault provider: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		p := extractData(body)
		pid, _ := p["id"].(string)
		name, _ := p["name"].(string)
		typ, _ := p["type"].(string)
		created, _ := p["created_at"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "  id:         %s\n", pid)
		fmt.Fprintf(cmd.OutOrStdout(), "  name:       %s\n", name)
		fmt.Fprintf(cmd.OutOrStdout(), "  type:       %s\n", typ)
		fmt.Fprintf(cmd.OutOrStdout(), "  created_at: %s\n", created)
		return nil
	},
}

func init() {
	addJSONFlag(vaultProviderShowCmd)
	vaultProviderCmd.AddCommand(vaultProviderShowCmd)
}
