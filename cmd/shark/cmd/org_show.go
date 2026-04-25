package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var orgShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/organizations/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("organization not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show organization: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		o := extractData(body)
		oid, _ := o["id"].(string)
		name, _ := o["name"].(string)
		slug, _ := o["slug"].(string)
		created, _ := o["created_at"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "  id:         %s\n", oid)
		fmt.Fprintf(cmd.OutOrStdout(), "  name:       %s\n", name)
		fmt.Fprintf(cmd.OutOrStdout(), "  slug:       %s\n", slug)
		fmt.Fprintf(cmd.OutOrStdout(), "  created_at: %s\n", created)
		return nil
	},
}

func init() {
	addJSONFlag(orgShowCmd)
	orgCmd.AddCommand(orgShowCmd)
}
