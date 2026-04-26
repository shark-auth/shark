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

// TestServeConfigFlagRemoved verifies that --config is no longer a registered
// flag on the serve command (Phase H hard removal).
func TestServeConfigFlagRemoved(t *testing.T) {
	f := serveCmd.Flags().Lookup("config")
	if f != nil {
		t.Errorf("--config flag is still registered on serveCmd; it should have been removed in Phase H")
	}
}

// TestServeConfigFlagParseError verifies that parsing --config as a CLI
// argument returns an unknown-flag error (the cobra flag set rejects it).
func TestServeConfigFlagParseError(t *testing.T) {
	// Reset any state from previous runs.
	serveCmd.ResetFlags()
	// Re-register only the flags that serve.go init() still sets up (no --config).
	serveCmd.Flags().StringVar(&serveProxyUpstream, "proxy-upstream", "", "")
	serveCmd.Flags().BoolVar(&serveNoPrompt, "no-prompt", false, "")

	err := serveCmd.Flags().Parse([]string{"--config", "sharkauth.yaml"})
	if err == nil {
		t.Fatal("expected error parsing --config flag, got nil")
	}
	if !strings.Contains(err.Error(), "unknown flag") && !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestServeDevFlagParseError verifies that parsing --dev as a CLI argument
// returns an unknown-flag error (the cobra flag set rejects it).
func TestServeDevFlagParseError(t *testing.T) {
	// Reset any state from previous runs.
	serveCmd.ResetFlags()
	// Re-register only the flags that serve.go init() still sets up (no --config, no --dev).
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
