// Package cmd â€” `shark agent register` subcommand (Lane E, E5).
//
// Uses the existing POST /api/v1/agents admin endpoint (handleCreateAgent).
// Spec directive: if no agent endpoints exist, scope-check with operator.
// The endpoint does exist (internal/api/agent_handlers.go), so we bind to it.
package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/shark-auth/shark/internal/cli"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agent identities",
}

var agentRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new agent identity",
	Long: `Creates a new agent identity via POST /api/v1/agents.
Returns the agent ID and client secret â€” the secret is shown only once.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		app, _ := cmd.Flags().GetString("app")
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		if name == "" {
			return maybeJSONErr(cmd, "invalid_args",
				fmt.Errorf("--name is required"))
		}

		payload := map[string]any{
			"name": name,
		}
		if description != "" {
			payload["description"] = description
		}
		if app != "" {
			// app slug maps to metadata for later SDK correlation
			payload["metadata"] = map[string]any{"app_slug": app}
		}

		body, code, err := adminDo(cmd, "POST", "/api/v1/agents", payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusConflict {
			return maybeJSONErr(cmd, "conflict",
				fmt.Errorf("agent with this name already exists"))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "register_failed",
				fmt.Errorf("register agent: %s", apiError(body, code)))
		}

		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}

		data := extractData(body)
		agentID, _ := data["id"].(string)
		clientID, _ := data["client_id"].(string)
		secret, _ := data["client_secret"].(string)

		out := cmd.OutOrStdout()
		cli.PrintSuccess(out, "agent registered")
		lines := []string{
			fmt.Sprintf("id:        %s", agentID),
			fmt.Sprintf("client_id: %s", clientID),
		}
		if secret != "" {
			lines = append(lines, fmt.Sprintf("secret:    %s  (shown once â€” save it now)", secret))
		}
		cli.PrintBox(out, "Agent Credentials", lines)
		return nil
	},
}

func init() {
	agentRegisterCmd.Flags().String("app", "", "app slug to associate this agent with")
	agentRegisterCmd.Flags().String("name", "", "human-readable name for the agent (required)")
	agentRegisterCmd.Flags().String("description", "", "optional description")
	_ = agentRegisterCmd.MarkFlagRequired("name")
	addJSONFlag(agentRegisterCmd)
	agentCmd.AddCommand(agentRegisterCmd)
	root.AddCommand(agentCmd)
}
