package cmd

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

var (
	appCreateName      string
	appCreateCallbacks []string
	appCreateLogouts   []string
	appCreateOrigins   []string
)

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new OAuth application",
	Long: `Creates a new OAuth application via the admin API and prints the client_secret once.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if appCreateName == "" {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("--name is required"))
		}

		if err := validateCLIURLs(appCreateCallbacks); err != nil {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("invalid callback URL: %w", err))
		}
		if err := validateCLIURLs(appCreateLogouts); err != nil {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("invalid logout URL: %w", err))
		}
		if err := validateCLIURLs(appCreateOrigins); err != nil {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("invalid origin URL: %w", err))
		}

		callbacks := appCreateCallbacks
		if callbacks == nil {
			callbacks = []string{}
		}
		logouts := appCreateLogouts
		if logouts == nil {
			logouts = []string{}
		}
		origins := appCreateOrigins
		if origins == nil {
			origins = []string{}
		}

		reqBody := map[string]any{
			"name":                  appCreateName,
			"allowed_callback_urls": callbacks,
			"allowed_logout_urls":   logouts,
			"allowed_origins":       origins,
		}

		body, code, err := adminDo(cmd, "POST", "/api/v1/admin/apps", reqBody)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code != http.StatusCreated && code != http.StatusOK {
			return maybeJSONErr(cmd, "create_failed", fmt.Errorf("create application: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		id, _ := body["id"].(string)
		clientID, _ := body["client_id"].(string)
		secret, _ := body["client_secret"].(string)

		fmt.Println()
		fmt.Println("  ============================================================")
		fmt.Printf("    Application created: %s\n", appCreateName)
		fmt.Printf("    id:            %s\n", id)
		fmt.Printf("    client_id:     %s\n", clientID)
		if secret != "" {
			fmt.Printf("    client_secret: %s   (shown once — save it)\n", secret)
		}
		fmt.Println("  ============================================================")
		fmt.Println()
		return nil
	},
}

// validateCLIURLs validates a list of URLs for CLI input.
// Rejects dangerous schemes; allows http/https and custom mobile schemes.
func validateCLIURLs(urls []string) error {
	for _, raw := range urls {
		if err := validateCLIURL(raw); err != nil {
			return fmt.Errorf("%q: %w", raw, err)
		}
	}
	return nil
}

func validateCLIURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("missing scheme")
	}
	switch u.Scheme {
	case "javascript", "file", "data", "vbscript":
		return fmt.Errorf("scheme %q is not allowed", u.Scheme)
	}
	return nil
}

func init() {
	appCreateCmd.Flags().StringVar(&appCreateName, "name", "", "application name (required)")
	appCreateCmd.Flags().StringArrayVar(&appCreateCallbacks, "callback", nil, "allowed callback URL (repeatable)")
	appCreateCmd.Flags().StringArrayVar(&appCreateLogouts, "logout", nil, "allowed logout URL (repeatable)")
	appCreateCmd.Flags().StringArrayVar(&appCreateOrigins, "origin", nil, "allowed origin (repeatable)")
	addJSONFlag(appCreateCmd)
	appCmd.AddCommand(appCreateCmd)
}
