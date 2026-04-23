package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var appRotateCmd = &cobra.Command{
	Use:   "rotate-secret <id-or-client-id>",
	Short: "Rotate the client secret of an OAuth application",
	Long:  `Generates a new client secret. The old secret is immediately invalid.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idOrClientID := args[0]

		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = "sharkauth.yaml"
		}
		cfg, err := config.Load(configPath)
		if err != nil {
			return maybeJSONErr(cmd, "config_load_failed", fmt.Errorf("load config: %w", err))
		}

		store, err := storage.NewSQLiteStore(cfg.Storage.Path)
		if err != nil {
			return maybeJSONErr(cmd, "database_open_failed", fmt.Errorf("open database: %w", err))
		}
		defer store.Close()

		ctx := context.Background()
		app, err := lookupApp(ctx, store, idOrClientID)
		if err != nil {
			return maybeJSONErr(cmd, "app_not_found", err)
		}

		newSecret, newHash, newPrefix, err := generateCLISecret()
		if err != nil {
			return maybeJSONErr(cmd, "secret_generation_failed", fmt.Errorf("generate secret: %w", err))
		}

		if err := store.RotateApplicationSecret(ctx, app.ID, newHash, newPrefix); err != nil {
			return maybeJSONErr(cmd, "rotate_failed", fmt.Errorf("rotate secret: %w", err))
		}

		rotatedAt := time.Now().UTC().Format("2006-01-02T15:04:05Z")

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"id":         app.ID,
				"client_id":  app.ClientID,
				"secret":     newSecret,
				"rotated_at": rotatedAt,
			})
		}

		fmt.Println()
		fmt.Println("  WARNING: Old secret immediately invalid.")
		fmt.Println()
		fmt.Println("  ============================================================")
		fmt.Printf("    Secret rotated for: %s\n", app.Name)
		fmt.Printf("    client_id:          %s\n", app.ClientID)
		fmt.Printf("    client_secret:      %s   (shown once — save it)\n", newSecret)
		fmt.Println("  ============================================================")
		fmt.Println()
		return nil
	},
}

func init() {
	appRotateCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	addJSONFlag(appRotateCmd)
	appCmd.AddCommand(appRotateCmd)
}
