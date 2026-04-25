// Package cmd — `shark agent register` subcommand.
//
// By default uses RFC 7591 Dynamic Client Registration (POST /oauth/register)
// which requires no auth. Use --admin to hit the operator-key-protected admin
// endpoint POST /api/v1/agents instead.
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agent identities",
}

var agentRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new agent identity",
	Long: `Registers a new agent identity.

Default (--dcr, RFC 7591): POSTs to POST /oauth/register — no auth required.
With --admin: POSTs to POST /api/v1/agents using the operator key (SHARK_ADMIN_TOKEN).

The client_secret is shown only once — save it immediately.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		app, _ := cmd.Flags().GetString("app")
		useAdmin, _ := cmd.Flags().GetBool("admin")

		if name == "" {
			return maybeJSONErr(cmd, "invalid_args",
				fmt.Errorf("--name is required"))
		}

		if useAdmin {
			return agentRegisterAdmin(cmd, name, description, app)
		}
		return agentRegisterDCR(cmd, name, description, app)
	},
}

// agentRegisterDCR calls POST /oauth/register (RFC 7591 — no auth required).
func agentRegisterDCR(cmd *cobra.Command, name, description, app string) error {
	// Build RFC 7591 client metadata body.
	meta := map[string]any{
		"client_name": name,
		"grant_types": []string{"client_credentials"},
		"token_endpoint_auth_method": "client_secret_basic",
	}
	if description != "" {
		meta["client_uri"] = "" // not ideal but keeps compat; description is non-standard
		// RFC 7591 has no "description" field; embed in client_name comment style
		meta["client_name"] = name + " — " + description
	}
	if app != "" {
		// software_id is the closest RFC 7591 analogue to an app slug
		meta["software_id"] = app
	}

	body, err := json.Marshal(meta)
	if err != nil {
		return maybeJSONErr(cmd, "request_failed", err)
	}

	baseURL := resolveAdminURL(cmd)
	req, err := http.NewRequest("POST", baseURL+"/oauth/register", bytes.NewReader(body))
	if err != nil {
		return maybeJSONErr(cmd, "request_failed", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := adminClient.Do(req)
	if err != nil {
		return maybeJSONErr(cmd, "request_failed", fmt.Errorf("POST /oauth/register: %w", err))
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result map[string]any
	_ = json.Unmarshal(raw, &result)

	if resp.StatusCode >= 300 {
		return maybeJSONErr(cmd, "register_failed",
			fmt.Errorf("DCR failed (%d): %s", resp.StatusCode, apiError(result, resp.StatusCode)))
	}

	if jsonFlag(cmd) {
		return writeJSON(cmd.OutOrStdout(), result)
	}

	clientID, _ := result["client_id"].(string)
	secret, _ := result["client_secret"].(string)
	regToken, _ := result["registration_access_token"].(string)

	fmt.Fprintf(cmd.OutOrStdout(), "agent registered (DCR)\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  client_id:                  %s\n", clientID)
	if secret != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  client_secret:              %s  (shown once — save it now)\n", secret)
	}
	if regToken != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  registration_access_token:  %s  (shown once — save it now)\n", regToken)
	}
	return nil
}

// agentRegisterAdmin calls POST /api/v1/agents (operator-key protected).
func agentRegisterAdmin(cmd *cobra.Command, name, description, app string) error {
	payload := map[string]any{
		"name": name,
	}
	if description != "" {
		payload["description"] = description
	}
	if app != "" {
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

	fmt.Fprintf(cmd.OutOrStdout(), "agent registered (admin)\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  id:            %s\n", agentID)
	fmt.Fprintf(cmd.OutOrStdout(), "  client_id:     %s\n", clientID)
	if secret != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  client_secret: %s  (shown once — save it now)\n", secret)
	}
	return nil
}

func init() {
	agentRegisterCmd.Flags().String("name", "", "human-readable name for the agent (required)")
	agentRegisterCmd.Flags().String("description", "", "optional description")
	agentRegisterCmd.Flags().String("app", "", "app slug (DCR: software_id; admin: metadata.app_slug)")
	agentRegisterCmd.Flags().Bool("admin", false, "use admin endpoint (POST /api/v1/agents) instead of DCR")
	_ = agentRegisterCmd.MarkFlagRequired("name")
	addJSONFlag(agentRegisterCmd)
	agentCmd.AddCommand(agentRegisterCmd)
	root.AddCommand(agentCmd)
}
