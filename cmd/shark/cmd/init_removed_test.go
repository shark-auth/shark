package cmd

import (
	"strings"
	"testing"
)

// TestInitCommandRemoved verifies that "shark init" is no longer a registered
// subcommand (Phase H hard removal — replaced by interactive shark serve).
func TestInitCommandRemoved(t *testing.T) {
	// Check cobra command tree does not contain "init".
	for _, sub := range root.Commands() {
		if sub.Use == "init" || strings.HasPrefix(sub.Use, "init ") {
			t.Errorf("'shark init' is still registered; it should have been removed in Phase H")
		}
	}
}
