package cmd

import "github.com/spf13/cobra"

// appCmd is the parent for all `shark app` subcommands.
var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage OAuth applications",
	Long:  `Create, list, inspect, update, rotate secrets for, and delete registered OAuth applications.`,
}

func init() {
	root.AddCommand(appCmd)
}
