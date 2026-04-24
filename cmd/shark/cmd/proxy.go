package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// proxyDeprecationMessage is printed to stderr whenever the long-since
// standalone `shark proxy` command is invoked. Kept as a package-level
// string so tests can assert against it without duplicating the literal.
const proxyDeprecationMessage = `The 'shark proxy' standalone command is deprecated in v1.5.
The reverse proxy now runs inside 'shark serve' and is configured via the Admin API:
  POST /api/v1/admin/proxy/rules
  POST /api/v1/admin/proxy/start|stop|reload
See docs/proxy_v1_5/ for migration guidance.`

// proxyCmd is a deprecation stub. PROXYV1_5 folded the standalone proxy
// process into `shark serve`: the embedded proxy owns its lifecycle,
// rules live in the DB (managed via `/api/v1/admin/proxy/rules`), and
// the admin lifecycle surface (`/api/v1/admin/proxy/{start,stop,reload,
// status}`) replaces the CLI flag matrix.
//
// The command is still registered so existing CI scripts invoking
// `shark proxy` surface an actionable deprecation message instead of a
// generic "unknown command" error. Any subcommand / arg combination
// falls through to RunE which prints the message and exits with 2
// (conventional "usage error" exit code).
//
// Printing directly to os.Stderr + calling os.Exit is deliberate: the
// cobra machinery would otherwise treat the non-nil error as a usage
// error and print the command's full Long string, drowning out the
// migration hint. Exiting from RunE also means no subcommand's Run is
// ever reached (they all share this parent RunE).
var proxyCmd = &cobra.Command{
	Use:                "proxy",
	Short:              "(deprecated) — the proxy now runs inside 'shark serve'",
	DisableSuggestions: true,
	// SilenceErrors + SilenceUsage keep cobra from re-printing the
	// deprecation message as an error or appending the full Usage block.
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, proxyDeprecationMessage)
		os.Exit(2)
		return nil
	},
}

func init() {
	// Register the stub at the root so `shark proxy` still resolves and
	// prints the deprecation message rather than cobra's "unknown
	// command" error.
	root.AddCommand(proxyCmd)
}
