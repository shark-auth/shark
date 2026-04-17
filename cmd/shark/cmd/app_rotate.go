package cmd

import (
	"context"
	"fmt"

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

		newSecret, newHash, newPrefix, err := generateCLISecret()
		if err != nil {
			return fmt.Errorf("generate secret: %w", err)
		}

		if err := store.RotateApplicationSecret(ctx, app.ID, newHash, newPrefix); err != nil {
			return fmt.Errorf("rotate secret: %w", err)
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
	appCmd.AddCommand(appRotateCmd)
}
