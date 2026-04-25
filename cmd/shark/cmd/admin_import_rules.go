package cmd

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var adminImportRulesCmd = &cobra.Command{
	Use:   "import-rules <file.yaml>",
	Short: "One-shot import legacy proxy YAML rules",
	Long: `Reads a YAML rules file and imports it into the running shark instance via
POST /api/v1/admin/proxy/rules/import.

Prints the count of rules imported and any per-row errors returned by the server.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		data, err := os.ReadFile(filePath) //#nosec G304 -- operator-supplied path for one-shot import
		if err != nil {
			return fmt.Errorf("read file %q: %w", filePath, err)
		}

		reqBody := map[string]any{
			"yaml": string(data),
		}

		body, code, err := adminDo(cmd, "POST", "/api/v1/admin/proxy/rules/import", reqBody)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code != http.StatusOK && code != http.StatusCreated {
			return maybeJSONErr(cmd, "import_failed", fmt.Errorf("import rules: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		// Human-readable summary.
		if count, ok := body["imported"].(float64); ok {
			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d rule(s) from %s\n", int(count), filePath)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Import complete: %s\n", filePath)
		}

		// Surface per-row errors if present.
		if errs, ok := body["errors"]; ok {
			if errArr, ok := errs.([]any); ok && len(errArr) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Errors:")
				for i, e := range errArr {
					fmt.Fprintf(cmd.OutOrStdout(), "  [%d] %v\n", i+1, e)
				}
			}
		}

		return nil
	},
}

func init() {
	addJSONFlag(adminImportRulesCmd)
	adminCmd.AddCommand(adminImportRulesCmd)
}
