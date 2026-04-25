package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var ssoCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an SSO connection",
	Long:  `Creates a new SSO connection via POST /api/v1/sso/connections.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		typ, _ := cmd.Flags().GetString("type")
		domain, _ := cmd.Flags().GetString("domain")

		if name == "" {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("--name is required"))
		}
		if typ == "" {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("--type is required (oidc or saml)"))
		}

		payload := map[string]any{
			"name": name,
			"type": typ,
		}
		if domain != "" {
			payload["domain"] = domain
		}

		body, code, err := adminDo(cmd, "POST", "/api/v1/sso/connections", payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusConflict {
			return maybeJSONErr(cmd, "conflict",
				fmt.Errorf("SSO connection already exists for domain %q", domain))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "create_failed",
				fmt.Errorf("create SSO connection: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		c := extractData(body)
		id, _ := c["id"].(string)
		fmt.Fprintf(cmd.OutOrStdout(), "SSO connection created: %s (%s)\n", name, id)
		return nil
	},
}

func init() {
	ssoCreateCmd.Flags().String("name", "", "connection name (required)")
	ssoCreateCmd.Flags().String("type", "", "connection type: oidc or saml (required)")
	ssoCreateCmd.Flags().String("domain", "", "email domain for auto-routing")
	_ = ssoCreateCmd.MarkFlagRequired("name")
	_ = ssoCreateCmd.MarkFlagRequired("type")
	addJSONFlag(ssoCreateCmd)
	ssoCmd.AddCommand(ssoCreateCmd)
}
