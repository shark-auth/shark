package cmd

import "github.com/spf13/cobra"

// adminCmd is the parent for all `shark admin` subcommands.
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin operations",
	Long:  `Administrative operations such as config dump and runtime management.`,
}

func init() {
	root.AddCommand(adminCmd)
}
