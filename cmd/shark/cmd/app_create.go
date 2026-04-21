package cmd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var (
	appCreateName      string
	appCreateCallbacks []string
	appCreateLogouts   []string
	appCreateOrigins   []string
)

var appCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new OAuth application",
	Long:  `Creates a new OAuth application and prints the client_secret once.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if appCreateName == "" {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("--name is required"))
		}

		if err := validateCLIURLs(appCreateCallbacks); err != nil {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("invalid callback URL: %w", err))
		}
		if err := validateCLIURLs(appCreateLogouts); err != nil {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("invalid logout URL: %w", err))
		}
		if err := validateCLIURLs(appCreateOrigins); err != nil {
			return maybeJSONErr(cmd, "invalid_args", fmt.Errorf("invalid origin URL: %w", err))
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

		nid, err := gonanoid.New(21)
		if err != nil {
			return fmt.Errorf("generate client id: %w", err)
		}
		clientID := "shark_app_" + nid

		secret, secretHash, secretPrefix, err := generateCLISecret()
		if err != nil {
			return fmt.Errorf("generate secret: %w", err)
		}

		appNid, _ := gonanoid.New()
		now := time.Now().UTC()

		callbacks := appCreateCallbacks
		if callbacks == nil {
			callbacks = []string{}
		}
		logouts := appCreateLogouts
		if logouts == nil {
			logouts = []string{}
		}
		origins := appCreateOrigins
		if origins == nil {
			origins = []string{}
		}

		app := &storage.Application{
			ID:                  "app_" + appNid,
			Name:                appCreateName,
			ClientID:            clientID,
			ClientSecretHash:    secretHash,
			ClientSecretPrefix:  secretPrefix,
			AllowedCallbackURLs: callbacks,
			AllowedLogoutURLs:   logouts,
			AllowedOrigins:      origins,
			IsDefault:           false,
			Metadata:            map[string]any{},
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		ctx := context.Background()
		if err := store.CreateApplication(ctx, app); err != nil {
			return maybeJSONErr(cmd, "create_failed", fmt.Errorf("create application: %w", err))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"app":    appToJSON(app),
				"secret": secret,
			})
		}

		fmt.Println()
		fmt.Println("  ============================================================")
		fmt.Printf("    Application created: %s\n", appCreateName)
		fmt.Printf("    id:            %s\n", app.ID)
		fmt.Printf("    client_id:     %s\n", clientID)
		fmt.Printf("    client_secret: %s   (shown once — save it)\n", secret)
		fmt.Println("  ============================================================")
		fmt.Println()
		return nil
	},
}

// validateCLIURLs validates a list of URLs for CLI input.
// Rejects dangerous schemes; allows http/https and custom mobile schemes.
func validateCLIURLs(urls []string) error {
	for _, raw := range urls {
		if err := validateCLIURL(raw); err != nil {
			return fmt.Errorf("%q: %w", raw, err)
		}
	}
	return nil
}

func validateCLIURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("missing scheme")
	}
	switch u.Scheme {
	case "javascript", "file", "data", "vbscript":
		return fmt.Errorf("scheme %q is not allowed", u.Scheme)
	}
	return nil
}

// generateCLISecret returns a base62-encoded 32-byte secret plus its hash and prefix.
func generateCLISecret() (secret, secretHash, secretPrefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	secret = cliBase62Encode(b)
	h := sha256.Sum256([]byte(secret))
	secretHash = hex.EncodeToString(h[:])
	secretPrefix = secret
	if len(secretPrefix) > 8 {
		secretPrefix = secretPrefix[:8]
	}
	return
}

const cliBase62Alpha = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func cliBase62Encode(b []byte) string {
	num := make([]byte, len(b))
	copy(num, b)
	var result []byte
	for !cliIsZero(num) {
		rem := cliDivmod(num, 62)
		result = append(result, cliBase62Alpha[rem])
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) == 0 {
		return "0"
	}
	return string(result)
}

func cliIsZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func cliDivmod(n []byte, d byte) byte {
	var rem uint64
	for i := range n {
		cur := rem*256 + uint64(n[i])
		n[i] = byte(cur / uint64(d)) //#nosec G115 -- base-62 long division: d is a byte (≤255) and cur/d fits in a byte by construction
		rem = cur % uint64(d)
	}
	return byte(rem)
}

func init() {
	appCreateCmd.Flags().StringVar(&appCreateName, "name", "", "application name (required)")
	appCreateCmd.Flags().StringArrayVar(&appCreateCallbacks, "callback", nil, "allowed callback URL (repeatable)")
	appCreateCmd.Flags().StringArrayVar(&appCreateLogouts, "logout", nil, "allowed logout URL (repeatable)")
	appCreateCmd.Flags().StringArrayVar(&appCreateOrigins, "origin", nil, "allowed origin (repeatable)")
	appCreateCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	addJSONFlag(appCreateCmd)
	appCmd.AddCommand(appCreateCmd)
}
