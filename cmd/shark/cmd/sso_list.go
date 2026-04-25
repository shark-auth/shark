package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var ssoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SSO connections",
	Long:  `List all SSO connections via GET /api/v1/sso/connections.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/sso/connections", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "list_failed",
				fmt.Errorf("list SSO connections: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		conns := extractDataArray(body)
		if len(conns) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No SSO connections found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tDOMAIN\tENABLED")
		for _, c := range conns {
			id, _ := c["id"].(string)
			name, _ := c["name"].(string)
			typ, _ := c["type"].(string)
			domain, _ := c["domain"].(string)
			enabled, _ := c["enabled"].(bool)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\n", id, name, typ, domain, enabled)
		}
		return w.Flush()
	},
}

func init() {
	addJSONFlag(ssoListCmd)
	ssoCmd.AddCommand(ssoListCmd)
}
