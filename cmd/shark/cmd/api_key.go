package cmd

import "github.com/spf13/cobra"

// apiKeyCmd is the parent for all `shark api-key` subcommands.
var apiKeyCmd = &cobra.Command{
	Use:   "api-key",
	Short: "Manage API keys",
	Long:  `Create, list, rotate, and revoke admin API keys.`,
}

func init() {
	root.AddCommand(apiKeyCmd)
}
