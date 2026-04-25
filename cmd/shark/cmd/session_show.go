package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		// Sessions are listed via admin list with filter; individual fetch via admin list
		// falls back to a targeted GET on the admin sessions route.
		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/sessions?limit=1&session_id="+id, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("session not found: %s", id))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show session: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		sessions := extractDataArray(body)
		if len(sessions) == 0 {
			return fmt.Errorf("session not found: %s", id)
		}
		s := sessions[0]
		sid, _ := s["id"].(string)
		ip, _ := s["ip_address"].(string)
		created, _ := s["created_at"].(string)
		expires, _ := s["expires_at"].(string)
		method, _ := s["auth_method"].(string)

		fmt.Fprintf(cmd.OutOrStdout(), "  id:          %s\n", sid)
		fmt.Fprintf(cmd.OutOrStdout(), "  auth_method: %s\n", method)
		fmt.Fprintf(cmd.OutOrStdout(), "  ip_address:  %s\n", ip)
		fmt.Fprintf(cmd.OutOrStdout(), "  created_at:  %s\n", created)
		fmt.Fprintf(cmd.OutOrStdout(), "  expires_at:  %s\n", expires)
		return nil
	},
}

func init() {
	addJSONFlag(sessionShowCmd)
	sessionCmd.AddCommand(sessionShowCmd)
}
