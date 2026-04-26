package cmd

import (
	"strings"
	"testing"
)

// TestServeDevFlagRemoved verifies that --dev is no longer a registered flag on
// the serve command. Attempting to parse it must return an error.
func TestServeDevFlagRemoved(t *testing.T) {
	// serveCmd.Flags() must not contain "dev".
	f := serveCmd.Flags().Lookup("dev")
	if f != nil {
		t.Errorf("--dev flag is still registered on serveCmd; it should have been removed")
	}
}

// TestServeResetFlagRemoved verifies --reset is no longer registered.
func TestServeResetFlagRemoved(t *testing.T) {
	f := serveCmd.Flags().Lookup("reset")
	if f != nil {
		t.Errorf("--reset flag is still registered on serveCmd; it should have been removed")
	}
}

// TestServeDevFlagParseError verifies that parsing --dev as a CLI argument
// returns an unknown-flag error (the cobra flag set rejects it).
func TestServeDevFlagParseError(t *testing.T) {
	// Reset any state from previous runs.
	serveCmd.ResetFlags()
	// Re-register the flags that serve.go init() sets up.
	serveCmd.Flags().StringVar(&serveConfigPath, "config", "sharkauth.yaml", "")
	serveCmd.Flags().StringVar(&serveProxyUpstream, "proxy-upstream", "", "")
	serveCmd.Flags().BoolVar(&serveNoPrompt, "no-prompt", false, "")

	err := serveCmd.Flags().Parse([]string{"--dev"})
	if err == nil {
		t.Fatal("expected error parsing --dev flag, got nil")
	}
	if !strings.Contains(err.Error(), "unknown flag") && !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("unexpected error message: %v", err)
	}
}
