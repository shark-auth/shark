// Package cmd — `shark proxy` admin subcommand tree (Lane E, E1).
//
// Provides lifecycle control (start/stop/reload/status) and rule management
// (list/add/show/delete/import) over the admin HTTP API.
//
// The old proxy.go deprecation stub is preserved alongside this new tree.
// `shark proxy` (bare invocation) still prints the deprecation message for
// backward-compat; the sub-commands are added as children of a *new* command
// `proxyAdminCmd` registered as "proxy-admin". This way existing CI scripts
// using `shark proxy` get the deprecation notice, and new tooling uses
// the explicit `shark proxy-admin ...` surface.
//
// Wait — re-reading the spec: the spec says `shark proxy start` etc, which
// means the subcommands ARE under `shark proxy`. The current stub has RunE
// that calls os.Exit(2) unconditionally, which would fire before any child
// RunE. We solve this by: remove RunE from the stub when child commands are
// present (cobra's default behaviour of printing help is acceptable), BUT
// keep the deprecation message in Short. Actually we need to fully replace
// proxyCmd. Let's do that cleanly here.
package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/cli"
)

// init replaces the old deprecation-stub proxyCmd with a real command tree
// that exposes lifecycle + rules sub-commands backed by the admin API.
func init() {
	// Remove the old stub registered by proxy.go's init().
	// We rebuild proxyCmd in place here by adding child commands; the stub's
	// RunE is overridden below so bare `shark proxy` shows help instead of
	// calling os.Exit(2) which would swallow the subcommands.
	proxyCmd.RunE = nil
	proxyCmd.Run = nil
	// Clear the deprecation short-description and replace with real one.
	proxyCmd.Short = "Manage the embedded reverse proxy (lifecycle + rules)"
	proxyCmd.Long = `Control the embedded reverse-proxy lifecycle (start/stop/reload/status)
and manage DB-backed proxy rules via the admin HTTP API.

Authentication: set SHARK_ADMIN_TOKEN or pass --token.
Server URL:     set SHARK_URL or pass --url (default http://localhost:8080).`

	// lifecycle subcommands
	proxyCmd.AddCommand(proxyStartCmd)
	proxyCmd.AddCommand(proxyStopCmd)
	proxyCmd.AddCommand(proxyReloadCmd)
	proxyCmd.AddCommand(proxyStatusCmd)

	// rules sub-tree
	proxyCmd.AddCommand(proxyRulesCmd)
	proxyRulesCmd.AddCommand(proxyRulesListCmd)
	proxyRulesCmd.AddCommand(proxyRulesAddCmd)
	proxyRulesCmd.AddCommand(proxyRulesShowCmd)
	proxyRulesCmd.AddCommand(proxyRulesDeleteCmd)
	proxyRulesCmd.AddCommand(proxyRulesImportCmd)

	// flags for rules add
	proxyRulesAddCmd.Flags().String("path", "", "URL path pattern (required, e.g. /api/*)")
	proxyRulesAddCmd.Flags().String("require", "", "require predicate (e.g. tier:pro, authenticated)")
	proxyRulesAddCmd.Flags().String("allow", "", "allow predicate (only 'anonymous' accepted)")
	proxyRulesAddCmd.Flags().String("app", "", "app slug / id to scope this rule to")
	proxyRulesAddCmd.Flags().String("name", "", "human-readable label for the rule")
	proxyRulesAddCmd.Flags().String("id", "", "client-specified rule ID (upsert); exit 2 if conflict with different payload")
	proxyRulesAddCmd.Flags().StringSlice("methods", nil, "HTTP methods to match (default: any)")
	proxyRulesAddCmd.Flags().StringSlice("scopes", nil, "required OAuth scopes (AND'd with require)")
	proxyRulesAddCmd.Flags().Bool("enabled", true, "whether the rule is enabled (default true)")
	proxyRulesAddCmd.Flags().Int("priority", 0, "rule priority (higher wins)")
	proxyRulesAddCmd.Flags().String("tier-match", "", "tier required by this rule (alias for require=tier:<name>)")
	proxyRulesAddCmd.Flags().Bool("m2m", false, "restrict to machine-to-machine callers only")
	_ = proxyRulesAddCmd.MarkFlagRequired("path")

	addJSONFlag(proxyRulesListCmd)
	addJSONFlag(proxyRulesAddCmd)
	addJSONFlag(proxyRulesShowCmd)
	addJSONFlag(proxyStartCmd)
	addJSONFlag(proxyStopCmd)
	addJSONFlag(proxyReloadCmd)
	addJSONFlag(proxyStatusCmd)
}

// ---------------------------------------------------------------------------
// lifecycle
// ---------------------------------------------------------------------------

var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the embedded reverse proxy",
	RunE:  proxyLifecycleAction("start", "POST", "/api/v1/admin/proxy/start"),
}

var proxyStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the embedded reverse proxy",
	RunE:  proxyLifecycleAction("stop", "POST", "/api/v1/admin/proxy/stop"),
}

var proxyReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload proxy rules and listeners without downtime",
	RunE:  proxyLifecycleAction("reload", "POST", "/api/v1/admin/proxy/reload"),
}

var proxyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current proxy lifecycle status",
	RunE:  proxyLifecycleAction("status", "GET", "/api/v1/admin/proxy/lifecycle"),
}

// proxyLifecycleAction returns a RunE closure for lifecycle commands.
func proxyLifecycleAction(name, method, path string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, method, path, nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "proxy_action_failed",
				fmt.Errorf("proxy %s failed: %s", name, apiError(body, code)))
		}
		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}
		// Human-readable: pull state_str + last_error from nested data object.
		data := extractData(body)
		stateStr, _ := data["state_str"].(string)
		lastErr, _ := data["last_error"].(string)
		listeners, _ := data["listeners"].(float64)
		rulesLoaded, _ := data["rules_loaded"].(float64)
		out := cmd.OutOrStdout()
		cli.PrintSuccess(out, fmt.Sprintf("state: %s  listeners: %d  rules_loaded: %d",
			stateStr, int(listeners), int(rulesLoaded)))
		if lastErr != "" {
			cli.PrintWarning(out, "last_error: "+lastErr)
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// rules sub-tree
// ---------------------------------------------------------------------------

var proxyRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage DB-backed proxy rules",
}

var proxyRulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List proxy rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/proxy/rules/db", nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "list_failed",
				fmt.Errorf("list rules: %s", apiError(body, code)))
		}
		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}
		rules := extractDataArray(body)
		if len(rules) == 0 {
			cli.PrintWarning(cmd.OutOrStdout(), "No proxy rules found.")
			return nil
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tPATTERN\tREQUIRE\tAPP_ID\tENABLED\tPRIORITY")
		for _, r := range rules {
			id, _ := r["id"].(string)
			pattern, _ := r["pattern"].(string)
			require, _ := r["require"].(string)
			if require == "" {
				if allow, ok := r["allow"].(string); ok && allow != "" {
					require = "allow:" + allow
				}
			}
			appID, _ := r["app_id"].(string)
			enabled, _ := r["enabled"].(bool)
			priority, _ := r["priority"].(float64)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%d\n",
				id, pattern, require, appID, enabled, int(priority))
		}
		return w.Flush()
	},
}

var proxyRulesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create or upsert a proxy rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("path")
		require, _ := cmd.Flags().GetString("require")
		allow, _ := cmd.Flags().GetString("allow")
		app, _ := cmd.Flags().GetString("app")
		name, _ := cmd.Flags().GetString("name")
		id, _ := cmd.Flags().GetString("id")
		methods, _ := cmd.Flags().GetStringSlice("methods")
		scopes, _ := cmd.Flags().GetStringSlice("scopes")
		enabled, _ := cmd.Flags().GetBool("enabled")
		priority, _ := cmd.Flags().GetInt("priority")
		tierMatch, _ := cmd.Flags().GetString("tier-match")
		m2m, _ := cmd.Flags().GetBool("m2m")

		if require == "" && allow == "" {
			return maybeJSONErr(cmd, "invalid_args",
				fmt.Errorf("one of --require or --allow is required"))
		}
		if require != "" && allow != "" {
			return maybeJSONErr(cmd, "invalid_args",
				fmt.Errorf("--require and --allow are mutually exclusive"))
		}
		if name == "" {
			name = path
		}

		payload := map[string]any{
			"name":    name,
			"pattern": path,
			"require": require,
			"allow":   allow,
			"enabled": enabled,
		}
		if app != "" {
			payload["app_id"] = app
		}
		if len(methods) > 0 {
			payload["methods"] = methods
		}
		if len(scopes) > 0 {
			payload["scopes"] = scopes
		}
		if priority != 0 {
			payload["priority"] = priority
		}
		if tierMatch != "" {
			payload["tier_match"] = tierMatch
		}
		if m2m {
			payload["m2m"] = true
		}
		if id != "" {
			payload["id"] = id
		}

		body, code, err := adminDo(cmd, "POST", "/api/v1/admin/proxy/rules/db", payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		// 409 with same id = conflict with different payload (idempotency spec).
		if code == http.StatusConflict {
			cli.PrintError(os.Stderr, "conflict: rule with this id already exists with a different payload")
			os.Exit(2)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "add_failed",
				fmt.Errorf("add rule: %s", apiError(body, code)))
		}
		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}
		data := extractData(body)
		ruleID, _ := data["id"].(string)
		cli.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("created rule %s", ruleID))
		return nil
	},
}

var proxyRulesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a proxy rule by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "GET", "/api/v1/admin/proxy/rules/db/"+args[0], nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("rule %q not found", args[0]))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "show_failed",
				fmt.Errorf("show rule: %s", apiError(body, code)))
		}
		return writeJSON(cmd.OutOrStdout(), body)
	},
}

var proxyRulesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a proxy rule by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, code, err := adminDo(cmd, "DELETE", "/api/v1/admin/proxy/rules/db/"+args[0], nil)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code == http.StatusNotFound {
			return maybeJSONErr(cmd, "not_found",
				fmt.Errorf("rule %q not found", args[0]))
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "delete_failed",
				fmt.Errorf("delete rule: %s", apiError(body, code)))
		}
		cli.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("deleted rule %s", args[0]))
		return nil
	},
}

var proxyRulesImportCmd = &cobra.Command{
	Use:   "import <file.yaml>",
	Short: "Bulk-import proxy rules from a YAML file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return maybeJSONErr(cmd, "file_read_failed",
				fmt.Errorf("read %s: %w", args[0], err))
		}
		payload := map[string]any{"yaml": string(data)}
		body, code, err := adminDo(cmd, "POST", "/api/v1/admin/proxy/rules/import", payload)
		if err != nil {
			return maybeJSONErr(cmd, "request_failed", err)
		}
		if code >= 300 {
			return maybeJSONErr(cmd, "import_failed",
				fmt.Errorf("import rules: %s", apiError(body, code)))
		}
		if jsonFlag(cmd) {
			return writeJSON(cmd.OutOrStdout(), body)
		}
		imported, _ := body["imported"].(float64)
		errList, _ := body["errors"].([]any)
		out := cmd.OutOrStdout()
		if len(errList) > 0 {
			cli.PrintWarning(out, fmt.Sprintf("imported %d rule(s), %d error(s):", int(imported), len(errList)))
			for _, e := range errList {
				if em, ok := e.(map[string]any); ok {
					idx, _ := em["index"].(string)
					n, _ := em["name"].(string)
					msg, _ := em["error"].(string)
					fmt.Fprintf(out, "  [%s] %s: %s\n", idx, n, msg)
				}
			}
		} else {
			cli.PrintSuccess(out, fmt.Sprintf("imported %d rule(s)", int(imported)))
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// extractData extracts the "data" field from an API response. If the response
// itself is the data (no wrapper), returns body directly.
func extractData(body map[string]any) map[string]any {
	if d, ok := body["data"].(map[string]any); ok {
		return d
	}
	return body
}

// extractDataArray extracts "data" as a slice of maps from an API response.
func extractDataArray(body map[string]any) []map[string]any {
	raw, ok := body["data"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// openBrowser opens the given URL in the default system browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform %q — open %s manually", runtime.GOOS, url)
	}
	return cmd.Start()
}
