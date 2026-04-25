package cmd

import "github.com/spf13/cobra"

// sessionCmd is the parent for all `shark session` subcommands.
var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage user sessions",
	Long:  `List, inspect, and revoke user sessions via the admin API.`,
}

func init() {
	root.AddCommand(sessionCmd)
}
