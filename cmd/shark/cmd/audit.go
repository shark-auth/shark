package cmd

import "github.com/spf13/cobra"

// auditCmd is the parent for all `shark audit` subcommands.
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit log operations",
	Long:  `List and export audit logs via the admin API.`,
}

func init() {
	root.AddCommand(auditCmd)
}
