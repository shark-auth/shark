package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Long:  `Creates a new user via POST /api/v1/admin/users.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		name, _ := cmd.Flags().GetString("name")
		password, _ := cmd.Flags().GetString("password")

		if email == "" {
			return maybeJSONErr(cmd, "invalid_args",
				fmt.Errorf("--email is required"))
		}

		payload := map[string]any{
			"email": email,
		}
		if name != "" {
			payload["name"] = name
		}
		if password != "" {
			payload["password"] = password
		}

		body, code, err := adminDo(cmd, "POST", "/api/v1/admin/users", payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusConflict {
			return maybeJSONErr(cmd, "conflict",
				fmt.Errorf("user with email %q already exists", email))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "create_failed",
				fmt.Errorf("create user: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		u := extractData(body)
		uid, _ := u["id"].(string)
		userEmail, _ := u["email"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "user created\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  id:    %s\n", uid)
		fmt.Fprintf(cmd.OutOrStdout(), "  email: %s\n", userEmail)
		return nil
	},
}

func init() {
	userCreateCmd.Flags().String("email", "", "user email address (required)")
	userCreateCmd.Flags().String("name", "", "display name")
	userCreateCmd.Flags().String("password", "", "initial password (optional)")
	_ = userCreateCmd.MarkFlagRequired("email")
	addJSONFlag(userCreateCmd)
	userCmd.AddCommand(userCreateCmd)
}
