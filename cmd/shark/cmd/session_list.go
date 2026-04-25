package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active sessions",
	Long:  `List active sessions via GET /api/v1/admin/sessions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		userID, _ := cmd.Flags().GetString("user")
		limit, _ := cmd.Flags().GetInt("limit")

		path := fmt.Sprintf("/api/v1/admin/sessions?limit=%d", limit)
		if userID != "" {
			path += "&user_id=" + userID
		}

		body, code, err := adminDo(cmd, "GET", path, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "list_failed",
				fmt.Errorf("list sessions: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		sessions := extractDataArray(body)
		if len(sessions) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No active sessions found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tUSER\tIP\tCREATED\tEXPIRES")
		for _, s := range sessions {
			id, _ := s["id"].(string)
			userEmail := ""
			if u, ok := s["user"].(map[string]any); ok {
				userEmail, _ = u["email"].(string)
			}
			ip, _ := s["ip_address"].(string)
			created, _ := s["created_at"].(string)
			expires, _ := s["expires_at"].(string)
			if len(created) > 10 {
				created = created[:10]
			}
			if len(expires) > 10 {
				expires = expires[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, userEmail, ip, created, expires)
		}
		return w.Flush()
	},
}

func init() {
	sessionListCmd.Flags().String("user", "", "filter by user ID")
	sessionListCmd.Flags().Int("limit", 50, "maximum number of sessions to return")
	addJSONFlag(sessionListCmd)
	sessionCmd.AddCommand(sessionListCmd)
}
