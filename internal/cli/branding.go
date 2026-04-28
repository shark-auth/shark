// Package cli owns all terminal presentation for the shark binary.
// It provides branded output, a custom slog handler, and helper functions
// used by cobra subcommands.
package cli

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"golang.org/x/term"
)

// sharkGlyphASCII is the user-supplied shark silhouette built from musical
// notes (♫). Source: playbook/ascii.txt (2026-04-26).
const sharkGlyphASCII = `
                                    
  ▄▄▄▄▄                             
 ██▀▀▀▀█▄ █▄                          
 ▀██▄  ▄▀ ██          ▄     ▄▄      
   ▀██▄▄  ████▄ ▄▀▀█▄ ████▄ ██ ▄█▀  
 ▄   ▀██▄ ██ ██ ▄█▀██ ██    ████    
 ▀██████▀▄██ ██▄▀█▄██▄█▀   ▄██ ▀█▄  
                                    
                                                                                                       
`

// PrintHeader writes the branded SharkAuth header to out.
// It prints the ASCII glyph, product tagline, binary size, version,
// docs URL, and repo URL.
func PrintHeader(out io.Writer) {
	ver := resolveVersion()
	size := binarySize()

	fmt.Fprint(out, sharkGlyphASCII)
	fmt.Fprintln(out)

	if IsColorEnabled(out) {
		fmt.Fprintf(out, "%sSharkAuth%s — Open Source Auth for Agents and Humans\n",
			ansiCyan, ansiReset)
	} else {
		fmt.Fprintln(out, "SharkAuth — Open Source Auth for Agents and Humans")
	}

	fmt.Fprintf(out, "Binary: %s · Version: %s\n", size, ver)
	fmt.Fprintf(out, "Docs:   https://sharkauth.com/docs\n")
	fmt.Fprintf(out, "Repo:   https://github.com/shark-auth/shark\n")
}

// resolveVersion reads the build version from ldflags or debug.ReadBuildInfo.
// Returns "dev" as a fallback.
func resolveVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok &&
		info.Main.Version != "" &&
		info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// binarySize returns the size of the running binary formatted as "XX MB".
func binarySize() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	mb := float64(info.Size()) / (1024 * 1024)
	if mb < 1 {
		return fmt.Sprintf("%.0f KB", float64(info.Size())/1024)
	}
	return fmt.Sprintf("%.0f MB", mb)
}

// PrintAdminKeyBanner prints a wide, eye-catching banner with the full admin
// API key on first boot. This is the operator's ONE chance to copy it before
// it's hashed away. Setup URL prints below the key. Width 80 cols.
func PrintAdminKeyBanner(out io.Writer, adminKey, setupURL, keyFilePath string) {
	const bar = "════════════════════════════════════════════════════════════════════════════════"
	c := IsColorEnabled(out)
	yellow := func(s string) string {
		if c {
			return ansiYellow + s + ansiReset
		}
		return s
	}
	cyan := func(s string) string {
		if c {
			return ansiCyan + s + ansiReset
		}
		return s
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, yellow(bar))
	fmt.Fprintln(out, yellow("              ⚠  ADMIN API KEY — YOU WILL ONLY SEE THIS ONCE  ⚠"))
	fmt.Fprintln(out, yellow(bar))
	fmt.Fprintln(out)
	fmt.Fprintf(out, "    %s\n", cyan(adminKey))
	fmt.Fprintln(out)
	if setupURL != "" {
		fmt.Fprintf(out, "    Setup URL:  %s\n", setupURL)
	}
	if keyFilePath != "" {
		fmt.Fprintf(out, "    Saved to:   %s  (perms 0600 — delete after pickup)\n", keyFilePath)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, yellow(bar))
	fmt.Fprintln(out)
}

// IsColorEnabled reports whether ANSI color codes should be emitted to w.
// Color is disabled when:
//   - NO_COLOR env var is non-empty
//   - w is os.Stdout and stdout is not a TTY
//   - w is os.Stderr and stderr is not a TTY
//
// For other writers, color is disabled (safe default).
func IsColorEnabled(w io.Writer) bool {
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

// ANSI escape codes — only emitted when IsColorEnabled returns true.
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiFaint  = "\033[2m"
)

// colorize wraps s with ANSI color code c if color is enabled for w,
// otherwise returns s unchanged.
func colorize(w io.Writer, c, s string) string {
	if IsColorEnabled(w) {
		return c + s + ansiReset
	}
	return s
}

// PrintAdminConfigured prints the "admin configured" re-run banner.
// Shown on second+ shark serve when at least one active admin API key exists.
//
//	✓ admin configured · Dashboard: <dashURL> · Sign in with your admin key
func PrintAdminConfigured(dashURL string) {
	fmt.Printf("  ✓ admin configured · Dashboard: %s · Sign in with your admin key\n", dashURL)
	fmt.Println()
}

// PrintSetupPending prints the "setup pending" re-run banner.
// Shown when no admin key exists yet but the bootstrap key file is present.
//
//	⚠ setup pending · Dashboard: <dashURL> · Key in: <keyPath>
func PrintSetupPending(dashURL, keyPath string) {
	fmt.Printf("  ⚠ setup pending · Dashboard: %s · Key in: %s\n", dashURL, keyPath)
	fmt.Println()
}
