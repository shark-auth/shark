package cmd

import (
	"fmt"
	"net/http"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OAuth applications",
	Long:  "List OAuth applications registered in the running shark instance.\n\nRequires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).",
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/apps", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code != http.StatusOK {
			return maybeJSONErr(cmd, "list_failed", fmt.Errorf("list applications: %s", apiError(body, code)))
		}

		// The API returns { "apps": [...] } or an array directly.
		var apps []any
		if arr, ok := body["apps"]; ok {
			if a, ok := arr.([]any); ok {
				apps = a
			}
		} else if arr, ok := body["data"]; ok {
			if a, ok := arr.([]any); ok {
				apps = a
			}
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		if len(apps) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No applications found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCLIENT ID\tCREATED")
		for _, raw := range apps {
			a, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id, _ := a["id"].(string)
			name, _ := a["name"].(string)
			clientID, _ := a["client_id"].(string)
			createdAt, _ := a["created_at"].(string)
			if len(createdAt) > 10 {
				createdAt = createdAt[:10]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", id, name, clientID, createdAt)
		}
		return w.Flush()
	},
}

// summarizeURLs returns a compact representation of a URL list.
func summarizeURLs(urls []string) string {
	if len(urls) == 0 {
		return "(none)"
	}
	if len(urls) <= 2 {
		result := urls[0]
		if len(urls) == 2 {
			result += ", " + urls[1]
		}
		return result
	}
	return fmt.Sprintf("%d urls", len(urls))
}

// maybeJSONErr emits a JSON error to stderr if --json is set, and returns err unchanged.
func maybeJSONErr(cmd *cobra.Command, code string, err error) error {
	if jsonFlag(cmd) {
		return writeJSONError(cmd, code, err, nil)
	}
	return err
}

func init() {
	addJSONFlag(appListCmd)
	appCmd.AddCommand(appListCmd)
}
