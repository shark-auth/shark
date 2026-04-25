package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var appRotateCmd = &cobra.Command{
	Use:   "rotate-secret <id-or-client-id>",
	Short: "Rotate the client secret of an OAuth application",
	Long: `Generates a new client secret. The old secret is immediately invalid.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrClientID := args[0]

		// Resolve the app ID first (the rotate endpoint uses the internal ID).
		appBody, appCode, err := adminDo(cmd, "GET", "/api/v1/admin/apps/"+idOrClientID, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if appCode != http.StatusOK {
			return maybeJSONErr(cmd, "app_not_found", fmt.Errorf("get application: %s", apiError(appBody, appCode)))
		}

		appID, _ := appBody["id"].(string)
		appName, _ := appBody["name"].(string)
		clientID, _ := appBody["client_id"].(string)

		body, code, err := adminDo(cmd, "POST", "/api/v1/admin/apps/"+appID+"/rotate-secret", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code != http.StatusOK {
			return maybeJSONErr(cmd, "rotate_failed", fmt.Errorf("rotate secret: %s", apiError(body, code)))
		}

		newSecret, _ := body["client_secret"].(string)
		rotatedAt, _ := body["rotated_at"].(string)

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"id":         appID,
				"client_id":  clientID,
				"secret":     newSecret,
				"rotated_at": rotatedAt,
			})
		}

		fmt.Println()
		fmt.Println("  WARNING: Old secret immediately invalid.")
		fmt.Println()
		fmt.Println("  ============================================================")
		fmt.Printf("    Secret rotated for: %s\n", appName)
		fmt.Printf("    client_id:          %s\n", clientID)
		if newSecret != "" {
			fmt.Printf("    client_secret:      %s   (shown once — save it)\n", newSecret)
		}
		fmt.Println("  ============================================================")
		fmt.Println()
		return nil
	},
}

func init() {
	addJSONFlag(appRotateCmd)
	appCmd.AddCommand(appRotateCmd)
}
