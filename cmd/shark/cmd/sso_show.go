package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var ssoShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single SSO connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "GET", "/api/v1/sso/connections/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("SSO connection not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show SSO connection: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		c := extractData(body)
		cid, _ := c["id"].(string)
		name, _ := c["name"].(string)
		typ, _ := c["type"].(string)
		domain, _ := c["domain"].(string)
		enabled, _ := c["enabled"].(bool)
		created, _ := c["created_at"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "  id:         %s\n", cid)
		fmt.Fprintf(cmd.OutOrStdout(), "  name:       %s\n", name)
		fmt.Fprintf(cmd.OutOrStdout(), "  type:       %s\n", typ)
		fmt.Fprintf(cmd.OutOrStdout(), "  domain:     %s\n", domain)
		fmt.Fprintf(cmd.OutOrStdout(), "  enabled:    %v\n", enabled)
		fmt.Fprintf(cmd.OutOrStdout(), "  created_at: %s\n", created)
		return nil
	},
}

func init() {
	addJSONFlag(ssoShowCmd)
	ssoCmd.AddCommand(ssoShowCmd)
}
