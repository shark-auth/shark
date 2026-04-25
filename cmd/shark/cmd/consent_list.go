package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var consentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OAuth consents",
	Long: `List OAuth consents. Use --user <id> to filter by user (calls GET /api/v1/admin/oauth/consents).

Example:
  shark consents list --user usr_abc123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, _ := cmd.Flags().GetString("user")

		path := "/api/v1/admin/oauth/consents"
		if userID != "" {
			path += "?user_id=" + userID
		}

		body, code, err := adminDo(cmd, "GET", path, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "list_failed",
				fmt.Errorf("list consents: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		consents := extractDataArray(body)
		if len(consents) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No consents found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tUSER\tCLIENT\tSCOPES\tGRANTED")
		for _, c := range consents {
			id, _ := c["id"].(string)
			userID2, _ := c["user_id"].(string)
			clientID, _ := c["client_id"].(string)
			scopes, _ := c["scopes"].(string)
			granted, _ := c["granted_at"].(string)
			if len(granted) > 10 {
				granted = granted[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, userID2, clientID, scopes, granted)
		}
		return w.Flush()
	},
}

func init() {
	consentListCmd.Flags().StringP("user", "u", "", "filter by user ID")
	addJSONFlag(consentListCmd)
	consentCmd.AddCommand(consentListCmd)
}
