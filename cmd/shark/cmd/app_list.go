package cmd

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var appListCmd = &cobra.Command{
	Use:   "list",
	Short: "List OAuth applications",
	RunE: func(cmd *cobra.Command, args []string) error {
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
		apps, err := store.ListApplications(ctx, 100, 0)
		if err != nil {
			return maybeJSONErr(cmd, "list_failed", fmt.Errorf("list applications: %w", err))
		}

		if jsonFlag(cmd) {
			out := make([]map[string]any, 0, len(apps))
			for _, a := range apps {
				out = append(out, appToJSON(a))
			}
			return writeJSON(cmd.OutOrStdout(), out)
		}

		if len(apps) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No applications found.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCLIENT ID\tCALLBACKS\tCREATED")
		for _, a := range apps {
			callbackSummary := summarizeURLs(a.AllowedCallbackURLs)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				a.ID, a.Name, a.ClientID, callbackSummary,
				a.CreatedAt.UTC().Format("2006-01-02"),
			)
		}
		return w.Flush()
	},
}

func summarizeURLs(urls []string) string {
	if len(urls) == 0 {
		return "(none)"
	}
	if len(urls) == 1 {
		return urls[0]
	}
	if len(urls) <= 2 {
		return strings.Join(urls, ", ")
	}
	return fmt.Sprintf("%d urls", len(urls))
}

// appToJSON returns the canonical JSON shape for an Application used by the CLI.
func appToJSON(a *storage.Application) map[string]any {
	return map[string]any{
		"id":                    a.ID,
		"name":                  a.Name,
		"client_id":             a.ClientID,
		"client_secret_prefix":  a.ClientSecretPrefix,
		"is_default":            a.IsDefault,
		"allowed_callback_urls": a.AllowedCallbackURLs,
		"allowed_logout_urls":   a.AllowedLogoutURLs,
		"allowed_origins":       a.AllowedOrigins,
		"created_at":            a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		"updated_at":            a.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// maybeJSONErr emits a JSON error to stderr if --json is set, and returns err unchanged.
func maybeJSONErr(cmd *cobra.Command, code string, err error) error {
	if jsonFlag(cmd) {
		return writeJSONError(cmd, code, err, nil)
	}
	return err
}

func init() {
	appListCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	addJSONFlag(appListCmd)
	appCmd.AddCommand(appListCmd)
}
