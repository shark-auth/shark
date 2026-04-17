package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var appShowCmd = &cobra.Command{
	Use:   "show <id-or-client-id>",
	Short: "Show a single OAuth application",
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

		callbacksJSON, _ := json.MarshalIndent(app.AllowedCallbackURLs, "              ", "  ")
		logoutsJSON, _ := json.MarshalIndent(app.AllowedLogoutURLs, "              ", "  ")
		originsJSON, _ := json.MarshalIndent(app.AllowedOrigins, "              ", "  ")

		fmt.Printf("  id:            %s\n", app.ID)
		fmt.Printf("  name:          %s\n", app.Name)
		fmt.Printf("  client_id:     %s\n", app.ClientID)
		fmt.Printf("  secret_prefix: %s\n", app.ClientSecretPrefix)
		fmt.Printf("  is_default:    %v\n", app.IsDefault)
		fmt.Printf("  callbacks:     %s\n", string(callbacksJSON))
		fmt.Printf("  logouts:       %s\n", string(logoutsJSON))
		fmt.Printf("  origins:       %s\n", string(originsJSON))
		fmt.Printf("  created_at:    %s\n", app.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
		fmt.Printf("  updated_at:    %s\n", app.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"))
		return nil
	},
}

// lookupApp tries GetApplicationByID then GetApplicationByClientID.
func lookupApp(ctx context.Context, store storage.Store, idOrClientID string) (*storage.Application, error) {
	app, err := store.GetApplicationByID(ctx, idOrClientID)
	if err == nil {
		return app, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get application: %w", err)
	}
	app, err = store.GetApplicationByClientID(ctx, idOrClientID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("application not found: %s", idOrClientID)
	}
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	return app, nil
}

func init() {
	appShowCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	appCmd.AddCommand(appShowCmd)
}
