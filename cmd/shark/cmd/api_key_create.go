package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var apiKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	Long:  `Creates a new admin API key via POST /api/v1/api-keys. The key is shown only once.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("--name is required"))
		}

		payload := map[string]any{
			"name":   name,
			"scopes": []string{"*"},
		}

		body, code, err := adminDo(cmd, "POST", "/api/v1/api-keys", payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusConflict {
			return maybeJSONErr(cmd, "conflict",
				fmt.Errorf("API key named %q already exists", name))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "create_failed",
				fmt.Errorf("create API key: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		k := extractData(body)
		id, _ := k["id"].(string)
		key, _ := k["key"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "API key created\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  id:  %s\n", id)
		if key != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  key: %s  (shown once — save it now)\n", key)
		}
		return nil
	},
}

func init() {
	apiKeyCreateCmd.Flags().String("name", "", "name for the API key (required)")
	_ = apiKeyCreateCmd.MarkFlagRequired("name")
	addJSONFlag(apiKeyCreateCmd)
	apiKeyCmd.AddCommand(apiKeyCreateCmd)
}
