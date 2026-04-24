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

// TestProxyCommand_IsRegistered confirms the stub is still wired into
// the root command tree. If a future refactor drops the init() call,
// `shark proxy` would start emitting cobra's generic "unknown command"
// message instead of the migration hint — caught here.
func TestProxyCommand_IsRegistered(t *testing.T) {
	found := false
	for _, c := range root.Commands() {
		if c.Name() == "proxy" {
			found = true
			if !strings.Contains(strings.ToLower(c.Short), "deprecated") {
				t.Errorf("proxy command Short should mention deprecation, got %q", c.Short)
			}
			if c.RunE == nil {
				t.Errorf("proxy command must have a RunE stub so invocations do not fall through to cobra help")
			}
			break
		}
	}
	if !found {
		t.Errorf("proxy command not registered under root")
	}
}
