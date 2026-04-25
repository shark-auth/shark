package cmd

import "github.com/spf13/cobra"

// consentCmd is the parent for all `shark consents` subcommands.
var consentCmd = &cobra.Command{
	Use:   "consents",
	Short: "Manage OAuth consents",
	Long:  `List and revoke OAuth consents via the admin API.`,
}

func init() {
	root.AddCommand(consentCmd)
}
