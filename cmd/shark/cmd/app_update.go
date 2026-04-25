package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var (
	appUpdateName         string
	appUpdateAddCallbacks []string
	appUpdateRmCallbacks  []string
	appUpdateAddLogouts   []string
	appUpdateRmLogouts    []string
	appUpdateAddOrigins   []string
	appUpdateRmOrigins    []string
)

var appUpdateCmd = &cobra.Command{
	Use:   "update <id-or-client-id>",
	Short: "Update an OAuth application",
	Long: `Update an OAuth application's name and/or allowed URLs.

Requires a running shark server (--url / SHARK_URL) and admin token (--token / SHARK_ADMIN_TOKEN).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrClientID := args[0]

		// Validate all URLs being added.
		if err := validateCLIURLs(appUpdateAddCallbacks); err != nil {
			return fmt.Errorf("invalid callback URL: %w", err)
		}
		if err := validateCLIURLs(appUpdateAddLogouts); err != nil {
			return fmt.Errorf("invalid logout URL: %w", err)
		}
		if err := validateCLIURLs(appUpdateAddOrigins); err != nil {
			return fmt.Errorf("invalid origin URL: %w", err)
		}

		// Fetch the current app state to compute mutations client-side.
		appBody, appCode, err := adminDo(cmd, "GET", "/api/v1/admin/apps/"+idOrClientID, nil)
		if err != nil {
			return fmt.Errorf("get application: %w", err)
		}
		if appCode != http.StatusOK {
			return fmt.Errorf("get application: %s", apiError(appBody, appCode))
		}

		appID, _ := appBody["id"].(string)

		currentCallbacks := anySliceToStrings(appBody["allowed_callback_urls"])
		currentLogouts := anySliceToStrings(appBody["allowed_logout_urls"])
		currentOrigins := anySliceToStrings(appBody["allowed_origins"])

		newCallbacks := applyURLMutations(currentCallbacks, appUpdateAddCallbacks, appUpdateRmCallbacks)
		newLogouts := applyURLMutations(currentLogouts, appUpdateAddLogouts, appUpdateRmLogouts)
		newOrigins := applyURLMutations(currentOrigins, appUpdateAddOrigins, appUpdateRmOrigins)

		patch := map[string]any{
			"allowed_callback_urls": newCallbacks,
			"allowed_logout_urls":   newLogouts,
			"allowed_origins":       newOrigins,
		}
		if appUpdateName != "" {
			patch["name"] = appUpdateName
		}

		body, code, err := adminDo(cmd, "PATCH", "/api/v1/admin/apps/"+appID, patch)
		if err != nil {
			return fmt.Errorf("update application: %w", err)
		}
		if code != http.StatusOK {
			return fmt.Errorf("update application: %s", apiError(body, code))
		}

		updatedName, _ := body["name"].(string)
		fmt.Printf("Updated application %s (%s)\n", updatedName, appID)
		fmt.Printf("  callbacks: %v\n", anySliceToStrings(body["allowed_callback_urls"]))
		fmt.Printf("  logouts:   %v\n", anySliceToStrings(body["allowed_logout_urls"]))
		fmt.Printf("  origins:   %v\n", anySliceToStrings(body["allowed_origins"]))
		return nil
	},
}

// anySliceToStrings converts []any (from JSON decode) to []string.
func anySliceToStrings(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// applyURLMutations adds then removes URLs from a slice, preserving order and deduplicating.
func applyURLMutations(current, add, remove []string) []string {
	// Index removals for O(1) lookup.
	removeSet := make(map[string]bool, len(remove))
	for _, u := range remove {
		removeSet[u] = true
	}

	// Start with existing, minus removals.
	var result []string
	for _, u := range current {
		if !removeSet[u] {
			result = append(result, u)
		}
	}

	// Add new items (skip duplicates).
	existing := make(map[string]bool, len(result))
	for _, u := range result {
		existing[u] = true
	}
	for _, u := range add {
		if !existing[u] {
			result = append(result, u)
			existing[u] = true
		}
	}

	if result == nil {
		result = []string{}
	}
	return result
}

func init() {
	appUpdateCmd.Flags().StringVar(&appUpdateName, "name", "", "new application name")
	appUpdateCmd.Flags().StringArrayVar(&appUpdateAddCallbacks, "add-callback", nil, "add a callback URL")
	appUpdateCmd.Flags().StringArrayVar(&appUpdateRmCallbacks, "remove-callback", nil, "remove a callback URL")
	appUpdateCmd.Flags().StringArrayVar(&appUpdateAddLogouts, "add-logout", nil, "add a logout URL")
	appUpdateCmd.Flags().StringArrayVar(&appUpdateRmLogouts, "remove-logout", nil, "remove a logout URL")
	appUpdateCmd.Flags().StringArrayVar(&appUpdateAddOrigins, "add-origin", nil, "add an origin")
	appUpdateCmd.Flags().StringArrayVar(&appUpdateRmOrigins, "remove-origin", nil, "remove an origin")
	appCmd.AddCommand(appUpdateCmd)
}
