package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/config"
)

var modeCmd = &cobra.Command{
	Use:   "mode [dev|prod]",
	Short: "Get or set the active mode (dev|prod)",
	Long: `With no argument, prints the current mode.

  shark mode        — prints current mode
  shark mode dev    — sets mode to dev (takes effect on next server restart)
  shark mode prod   — sets mode to prod (takes effect on next server restart)

The mode is stored in ~/.shark/state. Use 'shark serve' to start the server
in the active mode, or POST /api/v1/admin/system/swap-mode to persist without
restarting.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Read and print current mode.
			mode, err := config.ReadModeState()
			if err != nil {
				return fmt.Errorf("read mode: %w", err)
			}
			return printModeOutput(cmd, mode)
		}

		newMode := args[0]
		if newMode != "dev" && newMode != "prod" {
			return fmt.Errorf("mode must be 'dev' or 'prod', got %q", newMode)
		}

		if err := config.WriteModeState(newMode); err != nil {
			return fmt.Errorf("write mode: %w", err)
		}

		return printModeOutput(cmd, newMode)
	},
}

func printModeOutput(cmd *cobra.Command, mode string) error {
	if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
		out, _ := json.Marshal(map[string]string{"mode": mode})
		fmt.Fprintln(os.Stdout, string(out))
		return nil
	}
	fmt.Println(mode)
	return nil
}

func init() {
	modeCmd.Flags().Bool("json", false, "output as JSON")
	root.AddCommand(modeCmd)
}
