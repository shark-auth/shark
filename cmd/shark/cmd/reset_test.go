package cmd

import (
	"testing"
)

// TestResetCmdRegistered verifies the reset command is registered on the root.
// Full integration tests live in internal/api/system_handlers_test.go.
func TestResetCmdRegistered(t *testing.T) {
	for _, c := range root.Commands() {
		if c.Use == "reset <dev|prod|key>" {
			return
		}
	}
	t.Fatal("reset command not registered on root")
}

// TestModeCmdRegistered verifies the mode command is registered on the root.
func TestModeCmdRegistered(t *testing.T) {
	for _, c := range root.Commands() {
		if c.Use == "mode [dev|prod]" {
			return
		}
	}
	t.Fatal("mode command not registered on root")
}
