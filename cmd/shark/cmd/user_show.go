package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var userShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "GET", "/api/v1/users/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "user_not_found",
				fmt.Errorf("user not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show user: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		u := extractData(body)
		uid, _ := u["id"].(string)
		email, _ := u["email"].(string)
		name, _ := u["name"].(string)
		verified, _ := u["email_verified"].(bool)
		created, _ := u["created_at"].(string)
		updated, _ := u["updated_at"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "  id:             %s\n", uid)
		fmt.Fprintf(cmd.OutOrStdout(), "  email:          %s\n", email)
		fmt.Fprintf(cmd.OutOrStdout(), "  name:           %s\n", name)
		fmt.Fprintf(cmd.OutOrStdout(), "  email_verified: %v\n", verified)
		fmt.Fprintf(cmd.OutOrStdout(), "  created_at:     %s\n", created)
		fmt.Fprintf(cmd.OutOrStdout(), "  updated_at:     %s\n", updated)
		return nil
	},
}

func init() {
	addJSONFlag(userShowCmd)
	userCmd.AddCommand(userShowCmd)
}
