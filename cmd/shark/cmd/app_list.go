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
			return fmt.Errorf("load config: %w", err)
		}

		store, err := storage.NewSQLiteStore(cfg.Storage.Path)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer store.Close()

		ctx := context.Background()
		apps, err := store.ListApplications(ctx, 100, 0)
		if err != nil {
			return fmt.Errorf("list applications: %w", err)
		}

		if len(apps) == 0 {
			fmt.Println("No applications found.")
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

func init() {
	appListCmd.Flags().String("config", "sharkauth.yaml", "path to config file")
	appCmd.AddCommand(appListCmd)
}
