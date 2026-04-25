package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var apiKeyRotateCmd = &cobra.Command{
	Use:   "rotate <id>",
	Short: "Rotate an API key",
	Long:  `Rotates an API key via POST /api/v1/api-keys/{id}/rotate. The new key is shown only once.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		body, code, err := adminDo(cmd, "POST", "/api/v1/api-keys/"+id+"/rotate", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("API key not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "rotate_failed",
				fmt.Errorf("rotate API key: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		k := extractData(body)
		key, _ := k["key"].(string)
		fmt.Fprintf(cmd.OutOrStdout(), "API key rotated: %s\n", id)
		if key != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  new key: %s  (shown once — save it now)\n", key)
		}
		return nil
	},
}

func init() {
	addJSONFlag(apiKeyRotateCmd)
	apiKeyCmd.AddCommand(apiKeyRotateCmd)
}
