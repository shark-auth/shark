package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

// resetConfirmPhrase must match the server-side constant.
const cliResetConfirmPhrase = "RESET PROD"

var (
	resetBaseURL string
	resetAPIKey  string
	resetJSON    bool
)

var resetCmd = &cobra.Command{
	Use:   "reset <dev|prod|key>",
	Short: "Reset a database or rotate the admin API key",
	Long: `Reset SharkAuth state via the admin API.

  shark reset dev   — wipe dev.db, regenerate secrets (no confirmation)
  shark reset prod  — wipe shark.db, regenerate secrets (requires typed phrase)
  shark reset key   — rotate the admin API key only

Flags:
  --url   base URL of the running SharkAuth server (default http://localhost:8080)
  --key   admin API key (defaults to SHARK_ADMIN_KEY env var)
  --json  print JSON output`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		if target != "dev" && target != "prod" && target != "key" {
			return fmt.Errorf("target must be dev, prod, or key — got %q", target)
		}

		apiKey := resetAPIKey
		if apiKey == "" {
			apiKey = os.Getenv("SHARK_ADMIN_KEY")
		}
		if apiKey == "" {
			return fmt.Errorf("admin API key required: pass --key or set SHARK_ADMIN_KEY")
		}

		body := map[string]string{"target": target}
		if target == "prod" {
			// Prompt the operator to type the confirmation phrase.
			fmt.Fprintf(os.Stderr, "Type %q to confirm production reset: ", cliResetConfirmPhrase)
			var input string
			if _, err := fmt.Scanln(&input); err != nil {
				return fmt.Errorf("read confirmation: %w", err)
			}
			body["confirmation"] = input
		}

		b, _ := json.Marshal(body)
		url := resetBaseURL + "/api/v1/admin/system/reset"
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close() //#nosec G307

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		if resp.StatusCode >= 400 {
			msg, _ := result["message"].(string)
			if msg == "" {
				msg, _ = result["error"].(string)
			}
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, msg)
		}

		if resetJSON {
			out, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(out))
			return nil
		}

		// Human-friendly output.
		if msg, ok := result["message"].(string); ok {
			fmt.Println(msg)
		}
		if key, ok := result["admin_key"].(string); ok {
			fmt.Printf("\nNew admin key: %s\n", key)
			fmt.Println("(shown once — store it now)")
		}
		return nil
	},
}

func init() {
	resetCmd.Flags().StringVar(&resetBaseURL, "url", "http://localhost:8080", "SharkAuth server base URL")
	resetCmd.Flags().StringVar(&resetAPIKey, "key", "", "admin API key (or set SHARK_ADMIN_KEY)")
	resetCmd.Flags().BoolVar(&resetJSON, "json", false, "output JSON")
	root.AddCommand(resetCmd)
}
