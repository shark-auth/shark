// Package cmd — `shark user tier` subcommand (Lane E, E4).
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/cli"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userTierCmd = &cobra.Command{
	Use:   "tier <user-id-or-email> <tier>",
	Short: "Set a user's tier (free or pro)",
	Long: `Sets the tier for a user via PATCH /api/v1/admin/users/{id}/tier.

If a user email is supplied instead of an ID, the user is looked up first via
GET /api/v1/users?search=<email>.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		userRef := args[0]
		tier := strings.ToLower(strings.TrimSpace(args[1]))

		if tier != "free" && tier != "pro" {
			return maybeJSONErr(cmd, "invalid_tier",
				fmt.Errorf("tier must be \"free\" or \"pro\", got %q", tier))
		}

		// Resolve email → ID if the ref looks like an email.
		userID, err := resolveUserID(cmd, userRef)
		if err != nil {
			return maybeJSONErr(cmd, "user_not_found", err)
		}

		body, code, err := adminDo(cmd, "PATCH",
			"/api/v1/admin/users/"+userID+"/tier",
			map[string]any{"tier": tier})
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("user %q not found", userRef))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "tier_set_failed",
				fmt.Errorf("set tier: %s", apiError(body, code)))
		}
		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}
		cli.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("set tier %s for user %s", tier, userID))
		return nil
	},
}

// resolveUserID returns userRef unchanged if it looks like an opaque ID,
// otherwise performs a search-by-email against the admin users API.
func resolveUserID(cmd *cobra.Command, userRef string) (string, error) {
	// Heuristic: if it contains '@' treat as email.
	if !strings.Contains(userRef, "@") {
		return userRef, nil
	}

	q := url.Values{}
	q.Set("search", userRef)
	q.Set("limit", "5")
	body, code, err := adminDo(cmd, "GET", "/api/v1/users?"+q.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("lookup user by email: %w", err)
	}
	if code >= 300 {
		return "", fmt.Errorf("lookup user by email: %s", apiError(body, code))
	}

	// The users list endpoint returns {"users": [...], "total": N}
	rawUsers, ok := body["users"]
	if !ok {
		// Try generic "data" wrapper too.
		rawUsers = body["data"]
	}
	data, _ := json.Marshal(rawUsers)
	var users []map[string]any
	if err := json.Unmarshal(data, &users); err != nil || len(users) == 0 {
		return "", fmt.Errorf("no user found with email %q", userRef)
	}
	// Find exact match (search is fuzzy, email should be exact).
	needle := strings.ToLower(strings.TrimSpace(userRef))
	for _, u := range users {
		if email, ok := u["email"].(string); ok && strings.ToLower(email) == needle {
			if id, ok := u["id"].(string); ok {
				return id, nil
			}
		}
	}
	// Fallback: first result.
	if id, ok := users[0]["id"].(string); ok {
		return id, nil
	}
	return "", fmt.Errorf("no user found with email %q", userRef)
}

func init() {
	addJSONFlag(userTierCmd)
	userCmd.AddCommand(userTierCmd)
	root.AddCommand(userCmd)
}
