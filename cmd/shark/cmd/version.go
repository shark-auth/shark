package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/cli"
)

// version is injected at build time via -ldflags "-X ...cmd.version=vX.Y.Z".
// Falls back to the module version embedded by `go install` via debug.ReadBuildInfo.
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the shark version",
	Run: func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		cli.PrintHeader(out)
		fmt.Fprintf(out, "\nVersion: %s\n", resolveVersion())
	},
}

func resolveVersion() string {
	if version != "dev" && version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
