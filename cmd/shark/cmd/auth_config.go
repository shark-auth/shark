package cmd

import "github.com/spf13/cobra"

// authCmd is the parent for all `shark auth` subcommands.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage auth configuration",
	Long:  `Inspect and manage the authentication configuration via the admin API.`,
}

func init() {
	root.AddCommand(authCmd)
}
