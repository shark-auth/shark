package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var userUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		payload := map[string]any{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			payload["name"] = v
		}
		if cmd.Flags().Changed("email") {
			v, _ := cmd.Flags().GetString("email")
			payload["email"] = v
		}
		if cmd.Flags().Changed("email-verified") {
			v, _ := cmd.Flags().GetBool("email-verified")
			payload["email_verified"] = v
		}

		if len(payload) == 0 {
			return fmt.Errorf("no fields to update — specify at least one of --name, --email, --email-verified")
		}

		body, code, err := adminDo(cmd, "PATCH", "/api/v1/users/"+id, payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "user_not_found",
				fmt.Errorf("user not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "update_failed",
				fmt.Errorf("update user: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		u := extractData(body)
		uid, _ := u["id"].(string)
		email, _ := u["email"].(string)
		fmt.Fprintf(cmd.OutOrStdout(), "user updated: %s (%s)\n", uid, email)
		return nil
	},
}

func init() {
	userUpdateCmd.Flags().String("name", "", "new display name")
	userUpdateCmd.Flags().String("email", "", "new email address")
	userUpdateCmd.Flags().Bool("email-verified", false, "set email verified state")
	addJSONFlag(userUpdateCmd)
	userCmd.AddCommand(userUpdateCmd)
}
