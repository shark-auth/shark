package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var (
	appUpdateName          string
	appUpdateAddCallbacks  []string
	appUpdateRmCallbacks   []string
	appUpdateAddLogouts    []string
	appUpdateRmLogouts     []string
	appUpdateAddOrigins    []string
	appUpdateRmOrigins     []string
)

var appUpdateCmd = &cobra.Command{
	Use:   "update <id-or-client-id>",
	Short: "Update an OAuth application",
	Args:  cobra.ExactArgs(1),
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

		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = "sharkauth.yaml"
		}
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		store, err := storage.NewSQLiteStore(cfg.Storage.Path)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer store.Close()

		ctx := context.Background()
		app, err := lookupApp(ctx, store, idOrClientID)
		if err != nil {
			return err
		}

		if appUpdateName != "" {
			app.Name = appUpdateName
		}

		app.AllowedCallbackURLs = applyURLMutations(app.AllowedCallbackURLs, appUpdateAddCallbacks, appUpdateRmCallbacks)
		app.AllowedLogoutURLs = applyURLMutations(app.AllowedLogoutURLs, appUpdateAddLogouts, appUpdateRmLogouts)
		app.AllowedOrigins = applyURLMutations(app.AllowedOrigins, appUpdateAddOrigins, appUpdateRmOrigins)

		if err := store.UpdateApplication(ctx, app); err != nil {
			return fmt.Errorf("update application: %w", err)
		}

		// Re-fetch to show updated state.
		updated, err := store.GetApplicationByID(ctx, app.ID)
		if err != nil {
			return fmt.Errorf("re-fetch application: %w", err)
		}

		fmt.Printf("Updated application %s (%s)\n", updated.Name, updated.ID)
		fmt.Printf("  callbacks: %v\n", updated.AllowedCallbackURLs)
		fmt.Printf("  logouts:   %v\n", updated.AllowedLogoutURLs)
		fmt.Printf("  origins:   %v\n", updated.AllowedOrigins)
		return nil
	},
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
	appUpdateCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	appCmd.AddCommand(appUpdateCmd)
}
