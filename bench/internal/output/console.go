// Package output renders bench results to stdout.
package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/sharkauth/bench/internal/scenario"
)

// PrintScenarioStart prints a "[i/n] name ........ " header line.
func PrintScenarioStart(idx, total int, name string) {
	dots := strings.Repeat(".", maxNamePad-len(name))
	if dots == "" {
		dots = " "
	}
	fmt.Printf("[%d/%d] %s %s ", idx, total, name, dots)
}

// PrintScenarioDone prints "done in 30.0s" + per-scenario detail line.
func PrintScenarioDone(r scenario.Result, dur time.Duration) {
	fmt.Printf("done in %.1fs\n", dur.Seconds())
	fmt.Printf("   ok=%d  err=%d  rps=%.1f  p50=%s  p95=%s  p99=%s\n",
		r.OK, r.Errors, r.Throughput,
		fmtDur(r.LatencyP50), fmtDur(r.LatencyP95), fmtDur(r.LatencyP99))
	if len(r.Extra) > 0 {
		fmt.Printf("   extra: %v\n", r.Extra)
	}
}

// PrintSummary prints the final summary table.
func PrintSummary(profile string, results []scenario.Result) {
	fmt.Printf("\n=== summary (profile: %s) ===\n", profile)
	header := fmt.Sprintf("%-26s | %-7s | %-6s | %-6s | %-7s | %s",
		"scenario", "rps", "p50", "p95", "p99", "err")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))
	for _, r := range results {
		fmt.Printf("%-26s | %7.1f | %6s | %6s | %7s | %d\n",
			truncate(r.Name, 26),
			r.Throughput,
			fmtDur(r.LatencyP50),
			fmtDur(r.LatencyP95),
			fmtDur(r.LatencyP99),
			r.Errors,
		)
	}
}

const maxNamePad = 28

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// fmtDur formats a duration as integer ms when ≥1ms, else µs.
func fmtDur(d time.Duration) string {
	if d <= 0 {
		return "0ms"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
