package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var healthURL string

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check a running shark instance via /healthz",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(healthURL + "/healthz")
		if err != nil {
			return fmt.Errorf("reach %s: %w", healthURL, err)
		}
		defer resp.Body.Close()

		var body map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&body)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unhealthy (status %d): %v", resp.StatusCode, body)
		}
		fmt.Printf("ok — %s\n", healthURL)
		return nil
	},
}

func init() {
	healthCmd.Flags().StringVar(&healthURL, "url", "http://localhost:8080", "base URL of the running shark instance")
}
