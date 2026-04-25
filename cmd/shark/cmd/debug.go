package cmd

import "github.com/spf13/cobra"

// debugCmd is the parent for all `shark debug` subcommands.
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debugging utilities",
	Long:  `Debugging utilities for inspecting tokens and sessions.`,
}

func init() {
	root.AddCommand(debugCmd)
}
