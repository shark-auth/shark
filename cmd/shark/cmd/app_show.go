package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var appShowCmd = &cobra.Command{
	Use:   "show <id-or-client-id>",
	Short: "Show a single OAuth application",
	Long: `Show details of a single OAuth application by ID or client_id.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrClientID := args[0]

		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/apps/"+idOrClientID, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code != http.StatusOK {
			return maybeJSONErr(cmd, "app_not_found", fmt.Errorf("get application: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		callbackURLs, _ := body["allowed_callback_urls"]
		logoutURLs, _ := body["allowed_logout_urls"]
		origins, _ := body["allowed_origins"]

		callbacksJSON, _ := json.MarshalIndent(callbackURLs, "              ", "  ")
		logoutsJSON, _ := json.MarshalIndent(logoutURLs, "              ", "  ")
		originsJSON, _ := json.MarshalIndent(origins, "              ", "  ")

		id, _ := body["id"].(string)
		name, _ := body["name"].(string)
		clientID, _ := body["client_id"].(string)
		secretPrefix, _ := body["client_secret_prefix"].(string)
		isDefault, _ := body["is_default"].(bool)
		createdAt, _ := body["created_at"].(string)
		updatedAt, _ := body["updated_at"].(string)

		fmt.Printf("  id:            %s\n", id)
		fmt.Printf("  name:          %s\n", name)
		fmt.Printf("  client_id:     %s\n", clientID)
		fmt.Printf("  secret_prefix: %s\n", secretPrefix)
		fmt.Printf("  is_default:    %v\n", isDefault)
		fmt.Printf("  callbacks:     %s\n", string(callbacksJSON))
		fmt.Printf("  logouts:       %s\n", string(logoutsJSON))
		fmt.Printf("  origins:       %s\n", string(originsJSON))
		fmt.Printf("  created_at:    %s\n", createdAt)
		fmt.Printf("  updated_at:    %s\n", updatedAt)
		return nil
	},
}

func init() {
	addJSONFlag(appShowCmd)
	appCmd.AddCommand(appShowCmd)
}
