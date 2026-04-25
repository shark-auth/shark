package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// RenderCLIList renders the full command tree rooted at rootCmd to out,
// grouped by top-level subcommand. This is called by Phase E's cli.go
// command for `shark cli` output.
//
// Output format (TTY):
//
//	shark — SharkAuth CLI
//
//	  serve             Start the server
//	  version           Print the shark version
//	  health            Check a running shark instance
//	  ...
//
//	  proxy
//	    proxy start     Start the embedded reverse proxy
//	    proxy stop      Stop the embedded reverse proxy
//	    ...
func RenderCLIList(out io.Writer, rootCmd *cobra.Command) {
	color := IsColorEnabled(out)

	header := rootCmd.Use + " — " + rootCmd.Short
	if color {
		header = ansiCyan + rootCmd.Use + ansiReset + " — " + rootCmd.Short
	}
	fmt.Fprintln(out, header)
	fmt.Fprintln(out)

	cmds := rootCmd.Commands()
	for _, cmd := range cmds {
		if cmd.Hidden {
			continue
		}

		subs := cmd.Commands()
		if len(subs) == 0 {
			// leaf command
			printCmdLine(out, color, "  ", cmd.Use, cmd.Short)
		} else {
			// group header
			printGroupHeader(out, color, cmd.Use)
			for _, sub := range subs {
				if sub.Hidden {
					continue
				}
				fullUse := cmd.Use + " " + sub.Use
				printCmdLine(out, color, "    ", fullUse, sub.Short)
			}
		}
	}
}

func printCmdLine(out io.Writer, color bool, indent, use, short string) {
	useCol := fmt.Sprintf("%-32s", use)
	if color {
		useCol = ansiGreen + fmt.Sprintf("%-32s", use) + ansiReset
	}
	fmt.Fprintf(out, "%s%s  %s\n", indent, useCol, short)
}

func printGroupHeader(out io.Writer, color bool, name string) {
	// Strip subcommand placeholders like "proxy <command>"
	bare := strings.Fields(name)[0]
	fmt.Fprintln(out)
	if color {
		fmt.Fprintf(out, "  %s%s%s\n", ansiYellow, bare, ansiReset)
	} else {
		fmt.Fprintf(out, "  %s\n", bare)
	}
}
