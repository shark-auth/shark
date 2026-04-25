package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var userDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		yes, _ := cmd.Flags().GetBool("yes")

		if !yes {
			fmt.Printf("Delete user %s? [y/N] ", id)
			r := bufio.NewReader(os.Stdin)
			line, _ := r.ReadString('\n')
			line = strings.TrimSpace(strings.ToLower(line))
			if line != "y" && line != "yes" {
				fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
				return nil
			}
		}

		body, code, err := adminDo(cmd, "DELETE", "/api/v1/users/"+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "user_not_found",
				fmt.Errorf("user not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "delete_failed",
				fmt.Errorf("delete user: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{"deleted": id})
		}

		fmt.Fprintf(cmd.OutOrStdout(), "deleted user %s\n", id)
		return nil
	},
}

func init() {
	userDeleteCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	addJSONFlag(userDeleteCmd)
	userCmd.AddCommand(userDeleteCmd)
}
