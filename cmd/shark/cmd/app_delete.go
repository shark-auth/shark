package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var appDeleteYes bool

var appDeleteCmd = &cobra.Command{
	Use:   "delete <id-or-client-id>",
	Short: "Delete an OAuth application",
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

		if app.IsDefault {
			fmt.Fprintln(os.Stderr, "cannot delete default application")
			os.Exit(1)
		}

		if !appDeleteYes {
			fmt.Printf("Delete application %q (%s)? [y/N] ", app.Name, app.ID)
			r := bufio.NewReader(os.Stdin)
			line, _ := r.ReadString('\n')
			line = strings.TrimSpace(strings.ToLower(line))
			if line != "y" && line != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := store.DeleteApplication(ctx, app.ID); err != nil {
			return fmt.Errorf("delete application: %w", err)
		}

		fmt.Printf("Deleted application %s (%s)\n", app.Name, app.ID)
		return nil
	},
}

func init() {
	appDeleteCmd.Flags().BoolVar(&appDeleteYes, "yes", false, "skip confirmation prompt")
	appDeleteCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	appCmd.AddCommand(appDeleteCmd)
}
