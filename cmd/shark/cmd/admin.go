package cmd

import "github.com/spf13/cobra"

// adminCmd is the parent for all `shark admin` subcommands.
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin server management commands",
	Long:  `Commands for managing a running shark instance via the admin HTTP API.`,
}

func init() {
	root.AddCommand(adminCmd)
}
