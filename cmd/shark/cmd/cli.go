package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cliCmd = &cobra.Command{
	Use:   "cli",
	Short: "Print grouped command listing",
	Long:  `Print a grouped summary of all shark CLI commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "SHARK CLI — all commands")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "INSTANCE")
		fmt.Fprintln(out, "  shark serve                    Run the server (interactive on first boot)")
		fmt.Fprintln(out, "  shark mode <prod|dev>          Switch active DB (hot drain, no restart)")
		fmt.Fprintln(out, "  shark reset <prod|dev|key>     Wipe DB / rotate admin key")
		fmt.Fprintln(out, "  shark health                   Check running instance")
		fmt.Fprintln(out, "  shark version                  Print version")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "ADMIN")
		fmt.Fprintln(out, "  shark admin config dump        Print live config as JSON")
		fmt.Fprintln(out, "  shark whoami                   Identify current admin token")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "RESOURCES")
		fmt.Fprintln(out, "  shark user <list|create|...>")
		fmt.Fprintln(out, "  shark app <list|create|...>")
		fmt.Fprintln(out, "  shark agent <list|create|...>")
		fmt.Fprintln(out, "  shark keys <generate|rotate>")
		fmt.Fprintln(out, "  shark branding <get|set>")
		fmt.Fprintln(out, "  shark paywall preview")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "UTILITY")
		fmt.Fprintln(out, "  shark cli                      This screen")
		fmt.Fprintln(out, "  shark completion <shell>       Shell autocompletion script")
		return nil
	},
}

func init() {
	root.AddCommand(cliCmd)
}
