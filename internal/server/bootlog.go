// Package server: boot-phase output helper.
//
// W1.8 Surface 1 — minimal scaffold for "✓ phase  summary" lines during
// `shark serve` startup. The current boot path uses slog.Info; this package
// provides a uniform marker syntax so subsequent waves can swap individual
// phases (database open, migrations, jwt keys, vault, smtp, http listen)
// over to BootPhase one at a time without a single big-bang refactor.
//
// Heartbeat (single live status line) and framework-log suppression are
// deferred to W+1; this file only ships the helper + the dashboard banner.
package server

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"golang.org/x/term"
)

// Phase represents one boot step. Use StartPhase / Done / Fail to render
// "→ name" while in flight, "✓ name  summary" on success, "✗ name  err" on
// failure. On non-TTY stdout (CI), color codes are suppressed.
type Phase struct {
	name string
	w    io.Writer
	tty  bool
}

func StartPhase(name string) *Phase {
	tty := term.IsTerminal(int(os.Stdout.Fd()))
	p := &Phase{name: name, w: os.Stdout, tty: tty}
	if tty {
		fmt.Fprintf(p.w, "  \033[2m→\033[0m %-16s\n", name)
	}
	return p
}

func (p *Phase) Done(summary string) {
	if p.tty {
		fmt.Fprint(p.w, "\033[A\033[2K")
		fmt.Fprintf(p.w, "  \033[32m✓\033[0m %-16s \033[2m%s\033[0m\n", p.name, summary)
		return
	}
	fmt.Fprintf(p.w, "  ✓ %-16s %s\n", p.name, summary)
}

func (p *Phase) Fail(err error) {
	if p.tty {
		fmt.Fprint(p.w, "\033[A\033[2K")
		fmt.Fprintf(p.w, "  \033[31m✗\033[0m %-16s %s\n", p.name, err.Error())
		return
	}
	fmt.Fprintf(p.w, "  ✗ %-16s %s\n", p.name, err.Error())
}

// PrintDashboardURL prints the post-boot banner with the admin dashboard URL
// and (on first boot) the setup key path. Two blank lines around the banner
// give it visual separation from the phase log above and the heartbeat below.
func PrintDashboardURL(port int, setupKeyPath string, firstBoot bool) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// CI / non-TTY: still print, but without ANSI emphasis.
		fmt.Printf("\nDashboard: http://localhost:%d/admin\n", port)
		if firstBoot && setupKeyPath != "" {
			fmt.Printf("Setup key: %s (one-time)\n", setupKeyPath)
		}
		return
	}
	bold := "\033[1m"
	dim := "\033[2m"
	reset := "\033[0m"
	fmt.Println()
	fmt.Printf("  %sDashboard%s   %shttp://localhost:%d/admin%s\n", dim, reset, bold, port, reset)
	if firstBoot && setupKeyPath != "" {
		fmt.Printf("  %sSetup key%s   %s%s%s  %s(one-time)%s\n", dim, reset, bold, setupKeyPath, reset, dim, reset)
	}
	fmt.Println()
}

// platformBootHint returns a one-line hint for the current platform, used
// when the boot fails before HTTP comes up so the operator has somewhere to
// look. Currently a stub — kept here so subsequent waves can light it up
// without a fresh import.
func platformBootHint() string {
	switch runtime.GOOS {
	case "windows":
		return "Check Windows Defender / firewall isn't blocking :8080."
	case "darwin":
		return "Check macOS app firewall isn't prompting on first launch."
	default:
		return "Check that :8080 isn't already in use (sudo lsof -i :8080)."
	}
}
