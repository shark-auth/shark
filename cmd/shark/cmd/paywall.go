// Package cmd — `shark paywall` subcommand (Lane E, E2).
package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var paywallCmd = &cobra.Command{
	Use:   "paywall",
	Short: "Paywall preview utilities",
}

var paywallPreviewCmd = &cobra.Command{
	Use:   "preview <slug>",
	Short: "Fetch and print the rendered paywall HTML for an app slug",
	Long: `Fetches the rendered paywall page from GET /paywall/{slug}?tier=...&return=...
and prints the HTML to stdout. Use --open to open in the default browser instead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		slug := args[0]
		tier, _ := cmd.Flags().GetString("tier")
		returnURL, _ := cmd.Flags().GetString("return")
		open, _ := cmd.Flags().GetBool("open")

		if tier == "" {
			return fmt.Errorf("--tier is required (e.g. --tier pro)")
		}

		q := url.Values{}
		q.Set("tier", tier)
		if returnURL != "" {
			q.Set("return", returnURL)
		}
		path := "/paywall/" + slug + "?" + q.Encode()

		if open {
			fullURL := resolveAdminURL(cmd) + path
			fmt.Fprintf(cmd.OutOrStdout(), "opening %s\n", fullURL)
			return openBrowser(fullURL)
		}

		body, code, contentType, err := adminDoRaw(cmd, "GET", path)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		if code >= 300 {
			return fmt.Errorf("paywall preview failed (HTTP %d): %s", code, string(body))
		}
		_ = contentType // print regardless
		fmt.Fprint(cmd.OutOrStdout(), string(body))
		return nil
	},
}

func init() {
	paywallPreviewCmd.Flags().String("tier", "", "required tier to preview (e.g. pro, free)")
	paywallPreviewCmd.Flags().String("return", "/", "return URL embedded in the upgrade CTA")
	paywallPreviewCmd.Flags().Bool("open", false, "open the paywall in the default browser instead of printing HTML")
	paywallCmd.AddCommand(paywallPreviewCmd)
	root.AddCommand(paywallCmd)
}
