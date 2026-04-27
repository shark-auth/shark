package scenario

import (
	"context"
	"os"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
	"github.com/sharkauth/bench/internal/metrics"
)

// SignupStorm hammers POST /api/v1/auth/signup with random emails.
type SignupStorm struct {
	DBPath string // optional: file path of sqlite DB to stat for db_size_mb
}

// NewSignupStorm constructs the scenario.
func NewSignupStorm(dbPath string) *SignupStorm { return &SignupStorm{DBPath: dbPath} }

func (s *SignupStorm) Name() string { return "signup_storm" }

func (s *SignupStorm) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	return nil
}

func (s *SignupStorm) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	hist := metrics.New()
	ok := metrics.NewCounter()
	errs := metrics.NewCounter()

	body := func(workerID int) (bool, time.Duration) {
		email := uniqueEmail("signup")
		payload := map[string]string{"email": email, "password": validBenchPassword}
		start := time.Now()
		r, err := c.Post(ctx, "/api/v1/auth/signup", payload, nil)
		lat := time.Since(start)
		if err != nil {
			return false, lat
		}
		// 2xx is success
		if r.Status >= 200 && r.Status < 300 {
			return true, lat
		}
		return false, lat
	}

	start := time.Now()
	runWorkers(opts.Concurrency, opts.Duration, hist, ok, errs, body)
	dur := time.Since(start)

	extra := map[string]any{}
	if s.DBPath != "" {
		if fi, err := os.Stat(s.DBPath); err == nil {
			extra["db_size_mb"] = float64(fi.Size()) / (1024.0 * 1024.0)
		}
	}
	return finalize(s.Name(), hist, ok, errs, dur, extra)
}

func (s *SignupStorm) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}
