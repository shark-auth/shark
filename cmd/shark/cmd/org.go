package cmd

import "github.com/spf13/cobra"

// orgCmd is the parent for all `shark org` subcommands.
var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage organizations",
	Long:  `Inspect and manage organizations via the admin API.`,
}

func init() {
	root.AddCommand(orgCmd)
}
