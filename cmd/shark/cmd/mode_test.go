package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sharkauth/sharkauth/internal/config"
)

func TestModeReadWrite(t *testing.T) {
	// Use a temp dir as the home so we don't pollute ~/.shark.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir) // Windows

	// Default should be "prod" when no state file exists.
	mode, err := config.ReadModeState()
	if err != nil {
		t.Fatalf("ReadModeState on missing file: %v", err)
	}
	if mode != "prod" {
		t.Fatalf("expected default mode 'prod', got %q", mode)
	}

	// Write dev.
	if err := config.WriteModeState("dev"); err != nil {
		t.Fatalf("WriteModeState dev: %v", err)
	}
	mode, err = config.ReadModeState()
	if err != nil {
		t.Fatalf("ReadModeState after write: %v", err)
	}
	if mode != "dev" {
		t.Fatalf("expected 'dev', got %q", mode)
	}

	// State file should exist in ~/.shark/state.
	statePath := filepath.Join(tmpDir, ".shark", "state")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	// Write prod.
	if err := config.WriteModeState("prod"); err != nil {
		t.Fatalf("WriteModeState prod: %v", err)
	}
	mode, _ = config.ReadModeState()
	if mode != "prod" {
		t.Fatalf("expected 'prod', got %q", mode)
	}
}

func TestModeInvalidIsIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// Write garbage to the state file.
	sharkDir := filepath.Join(tmpDir, ".shark")
	_ = os.MkdirAll(sharkDir, 0700)
	_ = os.WriteFile(filepath.Join(sharkDir, "state"), []byte(`{"mode":"garbage"}`), 0600)

	mode, err := config.ReadModeState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid modes should default to prod.
	if mode != "prod" {
		t.Fatalf("expected 'prod' for invalid state, got %q", mode)
	}
}
