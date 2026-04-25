package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit logs",
	Long: `Exports audit logs via POST /api/v1/audit-logs/export.
The server returns CSV by default. Use --output to write to a file.

Examples:
  shark audit export --format csv --since 2026-01-01
  shark audit export --format csv --since 2026-01-01 --output audit.csv`,
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		output, _ := cmd.Flags().GetString("output")

		// Build JSON payload. The audit export endpoint reads from and/or to in the body.
		payload := map[string]any{}
		if since != "" {
			payload["from"] = since
		}
		if until != "" {
			payload["to"] = until
		}

		baseURL := resolveAdminURL(cmd)
		token, err := resolveAdminToken(cmd)
		if err != nil {
			return err
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", baseURL+"/api/v1/audit-logs/export", bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := adminClient.Do(req)
		if err != nil {
			return fmt.Errorf("export request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("export failed (HTTP %d): %s", resp.StatusCode, string(body))
		}

		var out io.Writer = cmd.OutOrStdout()
		if output != "" {
			f, ferr := os.Create(output)
			if ferr != nil {
				return fmt.Errorf("create output file: %w", ferr)
			}
			defer f.Close()
			out = f
		}

		if _, err = io.Copy(out, resp.Body); err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		if output != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "audit logs exported to %s\n", output)
		}
		return nil
	},
}

func init() {
	auditExportCmd.Flags().String("since", "", "start date, RFC3339 or YYYY-MM-DD (e.g. 2026-01-01)")
	auditExportCmd.Flags().String("until", "", "end date, RFC3339 or YYYY-MM-DD")
	auditExportCmd.Flags().StringP("output", "o", "", "write CSV to file instead of stdout")
	auditCmd.AddCommand(auditExportCmd)
}
