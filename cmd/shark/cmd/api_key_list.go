package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var apiKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	Long:  `List all API keys via GET /api/v1/api-keys.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/api-keys", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "list_failed",
				fmt.Errorf("list API keys: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		keys := extractDataArray(body)
		if len(keys) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No API keys found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPREFIX\tCREATED")
		for _, k := range keys {
			id, _ := k["id"].(string)
			name, _ := k["name"].(string)
			prefix, _ := k["prefix"].(string)
			created, _ := k["created_at"].(string)
			if len(created) > 10 {
				created = created[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, name, prefix, created)
		}
		return w.Flush()
	},
}

func init() {
	addJSONFlag(apiKeyListCmd)
	apiKeyCmd.AddCommand(apiKeyListCmd)
}
