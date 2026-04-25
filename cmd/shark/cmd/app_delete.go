package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var appDeleteYes bool

var appDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-client-id>",
	Short: "Delete an OAuth application",
	Long: `Delete an OAuth application by ID or client_id.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrClientID := args[0]

		// Fetch the app first so we can show its name in the prompt and guard default apps.
		appBody, appCode, err := adminDo(cmd, "GET", "/api/v1/admin/apps/"+idOrClientID, nil)
		if err != nil {
			return fmt.Errorf("get application: %w", err)
		}
		if appCode != http.StatusOK {
			return fmt.Errorf("get application: %s", apiError(appBody, appCode))
		}

		isDefault, _ := appBody["is_default"].(bool)
		if isDefault {
			fmt.Fprintln(os.Stderr, "cannot delete default application")
			os.Exit(1)
		}

		appName, _ := appBody["name"].(string)
		appID, _ := appBody["id"].(string)

		if !appDeleteYes {
			fmt.Printf("Delete application %q (%s)? [y/N] ", appName, appID)
			r := bufio.NewReader(os.Stdin)
			line, _ := r.ReadString('\n')
			line = strings.TrimSpace(strings.ToLower(line))
			if line != "y" && line != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		body, code, err := adminDo(cmd, "DELETE", "/api/v1/admin/apps/"+appID, nil)
		if err != nil {
			return fmt.Errorf("delete application: %w", err)
		}
		if code != http.StatusOK && code != http.StatusNoContent {
			return fmt.Errorf("delete application: %s", apiError(body, code))
		}

		fmt.Printf("Deleted application %s (%s)\n", appName, appID)
		return nil
	},
}

func init() {
	appDeleteCmd.Flags().BoolVar(&appDeleteYes, "yes", false, "skip confirmation prompt")
	appCmd.AddCommand(appDeleteCmd)
}
