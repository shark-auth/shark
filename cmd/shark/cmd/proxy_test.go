package cmd

import (
	"strings"
	"testing"
)

// TestProxyCommand_DeprecationMessageContainsMigrationHint sanity-checks
// the constant printed to stderr so CI regressions that accidentally
// truncate the message surface immediately. Mirrors the substrings the
// orchestrator doc promises dashboards / CI banners can rely on.
//
// Why not exercise RunE directly: the stub's RunE calls os.Exit(2) by
// design — it's the deprecation exit convention (usage error). Invoking
// RunE under `go test` would tear down the test process. The content
// guard below plus the registration guard in
// TestProxyCommand_IsRegistered cover the observable behaviour without
// needing a subprocess dance.
func TestProxyCommand_DeprecationMessageContainsMigrationHint(t *testing.T) {
	wantSubstrings := []string{
		"deprecated",
		"shark serve",
		"/api/v1/admin/proxy/rules",
		"/api/v1/admin/proxy/start|stop|reload",
		"docs/proxy_v1_5/",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(proxyDeprecationMessage, want) {
			t.Errorf("proxyDeprecationMessage missing substring %q\nfull message:\n%s",
				want, proxyDeprecationMessage)
		}
	}
}

// TestProxyCommand_IsRegistered confirms the proxy command is still wired into
// the root command tree with a real subcommand tree (Lane E upgrade).
func TestProxyCommand_IsRegistered(t *testing.T) {
	found := false
	for _, c := range root.Commands() {
		if c.Name() == "proxy" {
			found = true
			// Lane E upgraded proxy from deprecation stub to a real command.
			// Verify key subcommands are present.
			subNames := make(map[string]bool)
			for _, sub := range c.Commands() {
				subNames[sub.Name()] = true
			}
			for _, want := range []string{"start", "stop", "reload", "status", "rules"} {
				if !subNames[want] {
					t.Errorf("proxy subcommand %q not registered; got %v", want, subNames)
				}
			}
			break
		}
	}
	if !found {
		t.Errorf("proxy command not registered under root")
	}
}
