package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shark-auth/shark/internal/cli"
	"github.com/shark-auth/shark/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the shark version and build info",
	Run: func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		cli.PrintHeader(out)
		fmt.Fprintf(out, "\nVersion:    %s\n", version.Version)
		fmt.Fprintf(out, "Git Commit: %s\n", version.Commit)
		fmt.Fprintf(out, "Build Time: %s\n", version.BuildTime)
	},
}
