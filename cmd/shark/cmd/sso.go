package cmd

import "github.com/spf13/cobra"

// ssoCmd is the parent for all `shark sso` subcommands.
var ssoCmd = &cobra.Command{
	Use:   "sso",
	Short: "Manage SSO connections",
	Long:  `Create, list, inspect, update, and delete SSO (SAML/OIDC) connections.`,
}

func init() {
	root.AddCommand(ssoCmd)
}
