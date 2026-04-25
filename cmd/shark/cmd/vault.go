package cmd

import "github.com/spf13/cobra"

// vaultCmd is the parent for all `shark vault` subcommands.
var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage vault providers",
	Long:  `Inspect vault provider configurations via the admin API.`,
}

// vaultProviderCmd groups vault provider subcommands.
var vaultProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage vault providers",
}

func init() {
	vaultCmd.AddCommand(vaultProviderCmd)
	root.AddCommand(vaultCmd)
}
