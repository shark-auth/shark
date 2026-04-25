// Package cmd — `shark whoami` subcommand (Lane E, E6).
//
// Introspects the current admin token by calling GET /api/v1/auth/me
// (requires a user session Bearer, not an admin key) or the admin stats
// endpoint to verify the token is recognised. Since the spec says "calls
// existing userinfo endpoint", we try GET /api/v1/auth/me with the token
// as a Bearer and fall back to verifying the admin key via /admin/stats.
package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/cli"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Introspect the current token and print identity info",
	Long: `Calls GET /api/v1/auth/me with the configured token and prints
user/agent identity, tier, scopes, and associated app.

If the token is an admin API key (not a user JWT), the endpoint will return
401; in that case whoami reports the token type as "admin_api_key".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/auth/me", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}

		if code == http.StatusUnauthorized {
			// The token may be an admin API key rather than a user JWT.
			// Verify by hitting /admin/stats which requires a valid admin key.
			statsBody, statsCode, statsErr := adminDo(cmd, "GET", "/api/v1/admin/stats", nil)
			if statsErr == nil && statsCode == http.StatusOK {
				if jsonFlag(cmd) {
					return writeJSON(cmd.OutOrStdout(), map[string]any{
						"token_type": "admin_api_key",
						"valid":      true,
						"stats":      statsBody,
					})
				}
				cli.PrintSuccess(cmd.OutOrStdout(), "token_type: admin_api_key (valid)")
				return nil
			}
			return maybeJSONErr(cmd, "unauthorized",
				fmt.Errorf("token is not valid (tried user JWT + admin API key)"))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "whoami_failed",
				fmt.Errorf("whoami: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		// Human-readable output.
		id, _ := body["id"].(string)
		email, _ := body["email"].(string)
		tier := extractNestedString(body, "metadata", "tier")
		fmt.Fprintf(cmd.OutOrStdout(), "id:    %s\n", id)
		if email != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "email: %s\n", email)
		}
		if tier != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "tier:  %s\n", tier)
		}
		return nil
	},
}

// extractNestedString walks the body map following the supplied keys and
// returns the string value if found, else empty string.
func extractNestedString(body map[string]any, keys ...string) string {
	cur := body
	for i, k := range keys {
		v, ok := cur[k]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			s, _ := v.(string)
			return s
		}
		next, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		cur = next
	}
	return ""
}

func init() {
	addJSONFlag(whoamiCmd)
	root.AddCommand(whoamiCmd)
}
