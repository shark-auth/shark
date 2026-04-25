package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// PrettyHandler is a custom slog.Handler that emits formatted log lines
// suitable for interactive terminal output.
//
// Format (TTY):
//
//	13:42:17 INFO  http  POST /api/v1/auth/login              200  43ms
//
// Format (non-TTY): plain text key=value pairs (parseable by log aggregators).
type PrettyHandler struct {
	out   io.Writer
	level slog.Level
	mu    sync.Mutex
	attrs []slog.Attr // pre-attached attributes from WithAttrs
	group string      // current group prefix from WithGroup
}

// NewPrettyHandler creates a PrettyHandler writing to out at the given minimum level.
func NewPrettyHandler(out io.Writer, level slog.Level) *PrettyHandler {
	return &PrettyHandler{out: out, level: level}
}

// Enabled reports whether the handler handles records at level l.
func (h *PrettyHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

// Handle formats and writes the log record to the handler's output.
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.Enabled(context.Background(), r.Level) {
		return nil
	}

	isTTY := isTTYWriter(h.out)

	var buf bytes.Buffer

	if isTTY {
		h.formatTTY(&buf, r)
	} else {
		h.formatPlain(&buf, r)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf.Bytes())
	return err
}

// formatTTY writes a colored, column-aligned line to buf.
func (h *PrettyHandler) formatTTY(buf *bytes.Buffer, r slog.Record) {
	// Time: HH:MM:SS
	ts := r.Time.Format("15:04:05")
	fmt.Fprintf(buf, "%s%s%s ", ansiFaint, ts, ansiReset)

	// Level
	levelStr, levelColor := levelLabel(r.Level)
	fmt.Fprintf(buf, "%s%-5s%s ", levelColor, levelStr, ansiReset)

	// Collect all attributes (pre-attached + record's)
	attrs := collectAttrs(h.attrs, r)

	// Source column (key "source")
	source := extractAttr(attrs, "source")
	if source != "" {
		fmt.Fprintf(buf, "%s%-6s%s ", ansiCyan, source, ansiReset)
	}

	// Message
	fmt.Fprintf(buf, "%s", r.Message)

	// Remaining attrs (skip "source")
	extras := extraAttrs(attrs, "source")
	if len(extras) > 0 {
		fmt.Fprintf(buf, "  ")
		for i, a := range extras {
			if i > 0 {
				fmt.Fprintf(buf, "  ")
			}
			fmt.Fprintf(buf, "%s%s%s=%v", ansiFaint, a.Key, ansiReset, a.Value)
		}
	}

	buf.WriteByte('\n')
}

// formatPlain writes a plain-text key=value line to buf (for non-TTY / log aggregators).
func (h *PrettyHandler) formatPlain(buf *bytes.Buffer, r slog.Record) {
	ts := r.Time.Format(time.RFC3339)
	levelStr, _ := levelLabel(r.Level)
	fmt.Fprintf(buf, "time=%s level=%s msg=%q", ts, levelStr, r.Message)

	attrs := collectAttrs(h.attrs, r)
	for _, a := range attrs {
		fmt.Fprintf(buf, " %s=%v", a.Key, a.Value)
	}
	buf.WriteByte('\n')
}

// WithAttrs returns a new Handler whose attributes consist of both the
// receiver's attributes and attrs.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &PrettyHandler{
		out:   h.out,
		level: h.level,
		attrs: newAttrs,
		group: h.group,
	}
}

// WithGroup returns a new Handler with the given group name prepended.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	prefix := name
	if h.group != "" {
		prefix = h.group + "." + name
	}
	return &PrettyHandler{
		out:   h.out,
		level: h.level,
		attrs: h.attrs,
		group: prefix,
	}
}

// levelLabel returns the short label and ANSI color for a slog level.
func levelLabel(l slog.Level) (string, string) {
	switch {
	case l >= slog.LevelError:
		return "ERROR", ansiRed
	case l >= slog.LevelWarn:
		return "WARN", ansiYellow
	case l >= slog.LevelInfo:
		return "INFO", ansiCyan
	default:
		return "DEBUG", ansiFaint
	}
}

// collectAttrs merges pre-attached attrs with those on the record.
func collectAttrs(pre []slog.Attr, r slog.Record) []slog.Attr {
	all := make([]slog.Attr, len(pre))
	copy(all, pre)
	r.Attrs(func(a slog.Attr) bool {
		all = append(all, a)
		return true
	})
	return all
}

// extractAttr finds the string value of the first attr with the given key.
func extractAttr(attrs []slog.Attr, key string) string {
	for _, a := range attrs {
		if a.Key == key {
			return fmt.Sprintf("%v", a.Value.Any())
		}
	}
	return ""
}

// extraAttrs returns all attrs whose key is not in the skip set.
func extraAttrs(attrs []slog.Attr, skip ...string) []slog.Attr {
	skipSet := make(map[string]bool, len(skip))
	for _, s := range skip {
		skipSet[s] = true
	}
	out := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		if !skipSet[a.Key] {
			out = append(out, a)
		}
	}
	return out
}

// isTTYWriter reports whether w is a terminal.
func isTTYWriter(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	switch w {
	case os.Stdout:
		return term.IsTerminal(int(os.Stdout.Fd()))
	case os.Stderr:
		return term.IsTerminal(int(os.Stderr.Fd()))
	default:
		return false
	}
}

// NewServerLogger returns a configured slog.Logger for server use.
// When out is a TTY, it uses PrettyHandler; otherwise falls back to
// the standard text handler for log-aggregator parsability.
func NewServerLogger(out io.Writer, level slog.Level) *slog.Logger {
	if isTTYWriter(out) {
		return slog.New(NewPrettyHandler(out, level))
	}
	return slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: level}))
}

// levelColor applies the ANSI color for the given level string to s.
// Used by callers that format level labels themselves.
func levelColor(level, s string) string {
	switch strings.ToUpper(level) {
	case "ERROR":
		return ansiRed + s + ansiReset
	case "WARN":
		return ansiYellow + s + ansiReset
	case "INFO":
		return ansiCyan + s + ansiReset
	default:
		return ansiFaint + s + ansiReset
	}
}

// suppress unused-symbol linter warning for levelColor (used in tests / external callers)
var _ = levelColor
