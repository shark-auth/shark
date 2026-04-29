// Command bench is the SharkAuth black-box HTTP benchmark.
//
// Phase A wires three scenarios: signup_storm, login_burst, oauth_client_credentials.
// See playbook/16-benchmark-plan.md for the full spec.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
	"github.com/sharkauth/bench/internal/output"
	"github.com/sharkauth/bench/internal/scenario"
)

type profileDefaults struct {
	concurrency int
	duration    time.Duration
}

var profiles = map[string]profileDefaults{
	"smoke":     {concurrency: 10, duration: 30 * time.Second},
	"marketing": {concurrency: 200, duration: 60 * time.Second},
	"stress":    {concurrency: 500, duration: 5 * time.Minute},
	"cascade":   {concurrency: 1, duration: 60 * time.Second},
}

func main() {
	var (
		profile        = flag.String("profile", "smoke", "profile: smoke|marketing|stress|cascade")
		targetURL      = flag.String("target-url", "http://localhost:8080", "shark backend base URL")
		adminKeyFile   = flag.String("admin-key-file", filepath.Join("tests", "smoke", "data", "admin.key.firstboot"), "path to admin key file")
		duration       = flag.Duration("duration", 0, "per-scenario duration (overrides profile default)")
		concurrency    = flag.Int("concurrency", 0, "per-scenario worker count (overrides profile default)")
		outputPath     = flag.String("output", filepath.Join("bench", "REPORT.md"), "REPORT.md output path (Phase C)")
		baseline       = flag.String("baseline", "", "baseline.json path for regression diff (Phase C)")
		updateBaseline = flag.Bool("update-baseline", false, "rewrite bench/baseline.json after run (Phase C)")
		scenarioName   = flag.String("scenario", "", "run only this scenario by name")
		dpopMode       = flag.String("dpop-mode", "batched", "DPoP signing mode: batched|resign")
	)
	flag.Parse()

	def, ok := profiles[*profile]
	if !ok {
		fmt.Fprintf(os.Stderr, "bench: unknown profile %q (want smoke|marketing|stress|cascade)\n", *profile)
		os.Exit(2)
	}
	if *concurrency <= 0 {
		*concurrency = def.concurrency
	}
	if *duration <= 0 {
		*duration = def.duration
	}

	adminKey, err := loadAdminKey(*adminKeyFile)
	if err != nil {
		fmt.Printf("bench: could not load admin key from %s: %v\n", *adminKeyFile, err)
		fmt.Print("bench: enter admin api key manually: ")
		fmt.Scanln(&adminKey)
		adminKey = strings.TrimSpace(adminKey)
		if adminKey == "" {
			fmt.Fprintf(os.Stderr, "bench: admin key required to run benchmarks\n")
			os.Exit(1)
		}
	}

	fmt.Printf("bench: target=%s profile=%s concurrency=%d duration=%s dpop=%s\n",
		*targetURL, *profile, *concurrency, *duration, *dpopMode)
	fmt.Printf("bench: admin key ready (%d bytes)\n", len(adminKey))

	c := client.New(*targetURL, *concurrency)
	prover, err := client.NewProver(*dpopMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bench: dpop prover: %v\n", err)
		os.Exit(1)
	}
	c.SetProver(prover)

	fx := fixtures.NewBundle()

	// Resolve sqlite db path for db_size_mb extra in signup_storm. Best-effort.
	dbPath := filepath.Join("tests", "smoke", "data", "sharkauth.db")

	all := []scenario.Scenario{
		// Phase A (original)
		scenario.NewSignupStorm(dbPath),
		scenario.NewLoginBurst(100),
		scenario.NewOAuthClientCredentials(),
		// Phase B (marquee) — new
		scenario.NewTokenExchangeChain(),
		scenario.NewCascadeRevokeUserAgents(100),
		scenario.NewOAuthDPoP(),
		scenario.NewRBACPermissionCheckHot(),
		scenario.NewVaultReadConcurrent(),
	}

	scenarios := all
	if *scenarioName != "" {
		scenarios = filter(all, *scenarioName)
		if len(scenarios) == 0 {
			fmt.Fprintf(os.Stderr, "bench: no scenario named %q\n", *scenarioName)
			os.Exit(2)
		}
	}

	opts := scenario.Opts{
		Concurrency: *concurrency,
		Duration:    *duration,
		DPoPMode:    *dpopMode,
		AdminKey:    adminKey,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	results := make([]scenario.Result, 0, len(scenarios))
	for i, s := range scenarios {
		output.PrintScenarioStart(i+1, len(scenarios), s.Name())

		start := time.Now()
		setupCtx, setupCancel := context.WithTimeout(ctx, 2*time.Minute)
		if err := s.Setup(setupCtx, c, fx, opts); err != nil {
			setupCancel()
			fmt.Printf("\n   setup FAILED: %v\n", err)
			results = append(results, scenario.Result{
				Name:   s.Name(),
				Errors: 1,
				Extra:  map[string]any{"setup_error": err.Error()},
			})
			continue
		}
		setupCancel()

		runCtx, runCancel := context.WithTimeout(ctx, opts.Duration+30*time.Second)
		r := s.Run(runCtx, c, fx, opts)
		runCancel()

		_ = s.Teardown(ctx, c, fx)
		dur := time.Since(start)
		output.PrintScenarioDone(r, dur)
		results = append(results, r)
	}

	output.PrintSummary(*profile, results)

	// Phase A doesn't write REPORT.md or baseline; Phase C does. Acknowledge flags so users see them honoured.
	_ = outputPath
	_ = baseline
	_ = updateBaseline
}

func loadAdminKey(path string) (string, error) {
	bts, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bts)), nil
}

func filter(all []scenario.Scenario, name string) []scenario.Scenario {
	out := make([]scenario.Scenario, 0, 1)
	for _, s := range all {
		if s.Name() == name {
			out = append(out, s)
		}
	}
	return out
}
