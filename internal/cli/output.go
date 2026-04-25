package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// PrintSuccess writes a success indicator and message to out.
// With color: "✓ <msg>" in green; without: "+ <msg>".
func PrintSuccess(out io.Writer, msg string) {
	if IsColorEnabled(out) {
		fmt.Fprintf(out, "%s✓%s %s\n", ansiGreen, ansiReset, msg)
	} else {
		fmt.Fprintf(out, "+ %s\n", msg)
	}
}

// PrintWarning writes a warning indicator and message to out.
// With color: "! <msg>" in yellow; without: "! <msg>".
func PrintWarning(out io.Writer, msg string) {
	if IsColorEnabled(out) {
		fmt.Fprintf(out, "%s!%s %s\n", ansiYellow, ansiReset, msg)
	} else {
		fmt.Fprintf(out, "! %s\n", msg)
	}
}

// PrintError writes an error indicator and message to out.
// With color: "✗ <msg>" in red; without: "x <msg>".
func PrintError(out io.Writer, msg string) {
	if IsColorEnabled(out) {
		fmt.Fprintf(out, "%s✗%s %s\n", ansiRed, ansiReset, msg)
	} else {
		fmt.Fprintf(out, "x %s\n", msg)
	}
}

// PrintBox writes a bordered box containing title and lines to out.
// Uses Unicode box-drawing characters when the terminal supports UTF-8;
// otherwise falls back to ASCII +---+ framing.
//
// Example (UTF-8):
//
//	┌─ Admin API Key ────────────────┐
//	│  sk_live_xxxxxxxxxxxxxxxxxxxx  │
//	│  Use as: Authorization: ...    │
//	└────────────────────────────────┘
func PrintBox(out io.Writer, title string, lines []string) {
	if supportsUTF8(out) {
		printBoxUnicode(out, title, lines)
	} else {
		printBoxASCII(out, title, lines)
	}
}

// supportsUTF8 returns true when out is a TTY (which we assume can render
// UTF-8) or when LANG/LC_ALL contain "UTF-8" or "utf-8".
func supportsUTF8(out io.Writer) bool {
	// If it's not a TTY we play it safe and use ASCII.
	if !IsColorEnabled(out) {
		return false
	}
	return true
}

func printBoxUnicode(out io.Writer, title string, lines []string) {
	width := boxWidth(title, lines)

	// top border: ┌─ Title ────┐
	rightPad := width - utf8.RuneCountInString(title) - 4 // "─ " + " " + "─"
	if rightPad < 0 {
		rightPad = 0
	}
	fmt.Fprintf(out, "┌─ %s %s┐\n", title, strings.Repeat("─", rightPad))

	for _, line := range lines {
		pad := width - utf8.RuneCountInString(line) - 2
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(out, "│ %s%s│\n", line, strings.Repeat(" ", pad))
	}

	// bottom border
	fmt.Fprintf(out, "└%s┘\n", strings.Repeat("─", width))
}

func printBoxASCII(out io.Writer, title string, lines []string) {
	width := boxWidth(title, lines)

	// top border: +-- Title --+
	rightPad := width - len(title) - 4
	if rightPad < 0 {
		rightPad = 0
	}
	fmt.Fprintf(out, "+-- %s %s+\n", title, strings.Repeat("-", rightPad))

	for _, line := range lines {
		pad := width - len(line) - 2
		if pad < 0 {
			pad = 0
		}
		fmt.Fprintf(out, "| %s%s|\n", line, strings.Repeat(" ", pad))
	}

	fmt.Fprintf(out, "+%s+\n", strings.Repeat("-", width))
}

// boxWidth computes the inner width needed to fit the title and all lines,
// with a minimum of 40 characters. Includes 2 chars of padding (one each side).
func boxWidth(title string, lines []string) int {
	min := 40
	// title needs "─ " prefix and " ─" suffix (4 chars overhead for unicode, 4 for ascii)
	needed := utf8.RuneCountInString(title) + 6
	if needed > min {
		min = needed
	}
	for _, l := range lines {
		w := utf8.RuneCountInString(l) + 4 // "│ " + " │" overhead
		if w > min {
			min = w
		}
	}
	return min
}
