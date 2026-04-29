package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/shark-auth/shark/internal/cli"
)

var healthURL string

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check a running shark instance via /healthz",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(healthURL + "/healthz")
		if err != nil {
			return maybeJSONErr(cmd, "unreachable", fmt.Errorf("reach %s: %w", healthURL, err))
		}
		defer resp.Body.Close()

		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)

		if resp.StatusCode != http.StatusOK {
			err := fmt.Errorf("unhealthy (status %d): %v", resp.StatusCode, body)
			if jsonFlag(cmd) {
				return writeJSONError(cmd, "unhealthy", err, map[string]any{
					"status_code": resp.StatusCode,
					"body":        body,
				})
			}
			return err
		}

		if jsonFlag(cmd) {
			payload := map[string]any{
				"status": "ok",
				"url":    healthURL,
			}
			// Merge whatever fields /healthz returned (status, uptime_s, version, â€¦).
			for k, v := range body {
				payload[k] = v
			}
			return writeJSON(cmd.OutOrStdout(), payload)
		}

		cli.PrintSuccess(cmd.OutOrStdout(), "ok â€” "+healthURL)
		return nil
	},
}

func init() {
	healthCmd.Flags().StringVar(&healthURL, "url", "http://localhost:8080", "base URL of the running shark instance")
	addJSONFlag(healthCmd)
}
