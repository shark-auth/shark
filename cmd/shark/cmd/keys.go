package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	jwtmgr "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var keysRotate bool

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage signing keys",
}

var keysGenerateJWTCmd = &cobra.Command{
	Use:   "generate-jwt",
	Short: "Generate an RS256 JWT signing keypair and store it",
	Long: `Generates a 2048-bit RSA keypair for JWT signing (RS256).

Without --rotate: inserts a new active key. Fails if an active key already exists.
With --rotate: retires all current active keys and inserts a new active key.
Both old and new keys remain in the JWKS endpoint until the retired key's
rotated_at + 2*access_token_ttl has elapsed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			configPath = "sharkauth.yaml"
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.Server.Secret == "" {
			return fmt.Errorf("server.secret is not set in config")
		}

		store, err := storage.NewSQLiteStore(cfg.Storage.Path)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer store.Close()

		if err := storage.RunMigrations(store.DB(), migrationsFS, "migrations"); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}

		ctx := context.Background()
		mgr := jwtmgr.NewManager(&cfg.Auth.JWT, store, cfg.Server.BaseURL, cfg.Server.Secret)

		if keysRotate {
			if err := mgr.GenerateAndStore(ctx, true); err != nil {
				return fmt.Errorf("rotate signing key: %w", err)
			}
		} else {
			if err := mgr.GenerateAndStore(ctx, false); err != nil {
				return fmt.Errorf("generate signing key: %w", err)
			}
		}

		key, err := store.GetActiveSigningKey(ctx)
		if err != nil {
			return fmt.Errorf("get active key after generation: %w", err)
		}

		fmt.Printf("kid:       %s\n", key.KID)
		fmt.Printf("algorithm: %s\n", key.Algorithm)
		fmt.Printf("status:    %s\n", key.Status)
		if keysRotate {
			fmt.Println("Rotation complete. Old key(s) marked as retired.")
		} else {
			fmt.Println("New active signing key generated and stored.")
		}
		return nil
	},
}

func init() {
	keysGenerateJWTCmd.Flags().BoolVar(&keysRotate, "rotate", false, "retire active key(s) and generate a new one")
	// Inherit --config flag from parent context via PersistentFlags or use the global serve config
	keysGenerateJWTCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	keysCmd.AddCommand(keysGenerateJWTCmd)
}
