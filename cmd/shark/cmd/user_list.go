package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Long:  `List all users via GET /api/v1/users.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		search, _ := cmd.Flags().GetString("search")

		path := fmt.Sprintf("/api/v1/users?limit=%d", limit)
		if search != "" {
			path += "&search=" + search
		}

		body, code, err := adminDo(cmd, "GET", path, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "list_failed",
				fmt.Errorf("list users: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		users := extractDataArray(body)
		if len(users) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No users found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tEMAIL\tNAME\tVERIFIED\tCREATED")
		for _, u := range users {
			id, _ := u["id"].(string)
			email, _ := u["email"].(string)
			name, _ := u["name"].(string)
			verified, _ := u["email_verified"].(bool)
			created, _ := u["created_at"].(string)
			if len(created) > 10 {
				created = created[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", id, email, name, verified, created)
		}
		return w.Flush()
	},
}

func init() {
	userListCmd.Flags().Int("limit", 50, "maximum number of users to return")
	userListCmd.Flags().String("search", "", "filter by email or name")
	addJSONFlag(userListCmd)
	userCmd.AddCommand(userListCmd)
}

