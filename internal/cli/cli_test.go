package cli_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/shark-auth/shark/internal/cli"
)

// TestIsColorEnabled_NoColor verifies that NO_COLOR env var disables color.
func TestIsColorEnabled_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if cli.IsColorEnabled(os.Stdout) {
		t.Fatal("expected color disabled when NO_COLOR is set")
	}
}

// TestIsColorEnabled_NonTTY verifies that a plain bytes.Buffer is never
// considered a TTY (color always disabled).
func TestIsColorEnabled_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	if cli.IsColorEnabled(&buf) {
		t.Fatal("expected color disabled for non-TTY writer")
	}
}

// TestPrettyHandler_PlainOutput verifies that PrettyHandler writes parseable
// plain-text lines when the output is not a TTY (bytes.Buffer).
func TestPrettyHandler_PlainOutput(t *testing.T) {
	var buf bytes.Buffer
	h := cli.NewPrettyHandler(&buf, slog.LevelInfo)
	logger := slog.New(h)

	logger.Info("hello world", "source", "http")
	out := buf.String()

	if !strings.Contains(out, "msg=") {
		t.Errorf("expected msg= in plain output, got: %q", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.Contains(out, "source=") {
		t.Errorf("expected source= attr in output, got: %q", out)
	}
	// Must not contain ANSI escape sequences.
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI codes in non-TTY output, got: %q", out)
	}
}

// TestPrettyHandler_LevelFiltering verifies that records below the configured
// level are suppressed.
func TestPrettyHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	h := cli.NewPrettyHandler(&buf, slog.LevelWarn)
	logger := slog.New(h)

	logger.Info("this should be suppressed")
	logger.Warn("this should appear")

	out := buf.String()
	if strings.Contains(out, "suppressed") {
		t.Errorf("INFO message should have been filtered, got: %q", out)
	}
	if !strings.Contains(out, "this should appear") {
		t.Errorf("WARN message should appear, got: %q", out)
	}
}

// TestPrettyHandler_Enabled verifies Enabled() respects the level threshold.
func TestPrettyHandler_Enabled(t *testing.T) {
	h := cli.NewPrettyHandler(os.Stderr, slog.LevelError)
	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("INFO should not be enabled when threshold is ERROR")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("ERROR should be enabled when threshold is ERROR")
	}
}

// TestPrintSuccess_NoColor verifies plain output with NO_COLOR set.
func TestPrintSuccess_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	cli.PrintSuccess(&buf, "operation complete")
	out := buf.String()
	if !strings.Contains(out, "operation complete") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI codes, got: %q", out)
	}
}

// TestPrintBox_ASCII verifies PrintBox produces bordered output in a non-TTY context.
func TestPrintBox_ASCII(t *testing.T) {
	var buf bytes.Buffer
	cli.PrintBox(&buf, "Test Box", []string{"line one", "line two"})
	out := buf.String()
	if !strings.Contains(out, "Test Box") {
		t.Errorf("expected title in box output, got: %q", out)
	}
	if !strings.Contains(out, "line one") {
		t.Errorf("expected line one in box output, got: %q", out)
	}
}

// TestPrintHeader_NoColor verifies header output is clean text when NO_COLOR set.
func TestPrintHeader_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	cli.PrintHeader(&buf)
	out := buf.String()
	if !strings.Contains(out, "SharkAuth") {
		t.Errorf("expected SharkAuth in header, got: %q", out)
	}
	if strings.Contains(out, "\033[") {
		t.Errorf("expected no ANSI codes in NO_COLOR mode, got: %q", out)
	}
}
