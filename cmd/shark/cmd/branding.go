// Package cmd — `shark branding` subcommand (Lane E, E3).
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var brandingCmd = &cobra.Command{
	Use:   "branding",
	Short: "Manage app branding design tokens",
}

var brandingSetCmd = &cobra.Command{
	Use:   "set <slug>",
	Short: "Set design tokens for an app",
	Long: `Sets branding design tokens via PATCH /api/v1/admin/branding/design-tokens.

Tokens may be supplied as key=value pairs with --token (multiple allowed):

  shark branding set my-app --token colors.primary=#6366f1 --token typography.font_family=Inter

Or as a JSON file with --from-file:

  shark branding set my-app --from-file tokens.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args[0] // slug reserved for future per-app scoping; today the endpoint is global

		tokenPairs, _ := cmd.Flags().GetStringArray("token")
		fromFile, _ := cmd.Flags().GetString("from-file")

		var tokens map[string]any

		switch {
		case fromFile != "":
			data, err := os.ReadFile(fromFile)
			if err != nil {
				return maybeJSONErr(cmd, "file_read_failed",
					fmt.Errorf("read %s: %w", fromFile, err))
			}
			if err := json.Unmarshal(data, &tokens); err != nil {
				return maybeJSONErr(cmd, "invalid_json",
					fmt.Errorf("parse %s: %w", fromFile, err))
			}

		case len(tokenPairs) > 0:
			tokens = make(map[string]any)
			for _, kv := range tokenPairs {
				idx := strings.IndexByte(kv, '=')
				if idx < 0 {
					return maybeJSONErr(cmd, "invalid_token_pair",
						fmt.Errorf("token pair %q must be key=value", kv))
				}
				tokens[kv[:idx]] = kv[idx+1:]
			}

		default:
			return maybeJSONErr(cmd, "invalid_args",
				fmt.Errorf("provide --token key=value pairs or --from-file tokens.json"))
		}

		payload := map[string]any{"design_tokens": tokens}
		body, code, err := adminDo(cmd, "PATCH", "/api/v1/admin/branding/design-tokens", payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "branding_set_failed",
				fmt.Errorf("set branding tokens: %s", apiError(body, code)))
		}
		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "design tokens updated")
		return nil
	},
}

var brandingGetCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Read current branding for an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = args[0] // slug; today endpoint is global

		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/branding", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "branding_get_failed",
				fmt.Errorf("get branding: %s", apiError(body, code)))
		}
		return writeJSON(cmd.OutOrStdout(), body)
	},
}

func init() {
	brandingSetCmd.Flags().StringArray("token", nil, "design token as key=value (may be repeated)")
	brandingSetCmd.Flags().String("from-file", "", "path to a JSON file containing design tokens")
	addJSONFlag(brandingSetCmd)
	addJSONFlag(brandingGetCmd)
	brandingCmd.AddCommand(brandingSetCmd)
	brandingCmd.AddCommand(brandingGetCmd)
	root.AddCommand(brandingCmd)
}
