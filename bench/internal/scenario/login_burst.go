package scenario

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
	"github.com/sharkauth/bench/internal/metrics"
)

// LoginBurst pre-creates N users (default 100), then hammers POST /api/v1/auth/login.
type LoginBurst struct {
	NumFixtureUsers int
}

// NewLoginBurst constructs the scenario.
func NewLoginBurst(numUsers int) *LoginBurst {
	if numUsers <= 0 {
		numUsers = 100
	}
	return &LoginBurst{NumFixtureUsers: numUsers}
}

func (s *LoginBurst) Name() string { return "login_burst" }

func (s *LoginBurst) Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error {
	// Create NumFixtureUsers users serially-but-batched (concurrency=4 to keep bcrypt under control).
	const setupConcurrency = 4
	type job struct{ idx int }
	jobs := make(chan job, s.NumFixtureUsers)
	for i := 0; i < s.NumFixtureUsers; i++ {
		jobs <- job{idx: i}
	}
	close(jobs)

	errCh := make(chan error, setupConcurrency)
	for w := 0; w < setupConcurrency; w++ {
		go func() {
			for j := range jobs {
				email := fmt.Sprintf("loginfx-%s-%04d@bench.local", runID, j.idx)
				body := map[string]string{"email": email, "password": validBenchPassword}
				var resp struct {
					ID    string `json:"id"`
					Email string `json:"email"`
					Token string `json:"token"`
				}
				r, err := c.JSON(ctx, "POST", "/api/v1/auth/signup", body, nil, &resp)
				if err != nil {
					errCh <- fmt.Errorf("signup fixture %d: %w", j.idx, err)
					return
				}
				if r.Status < 200 || r.Status >= 300 {
					errCh <- fmt.Errorf("signup fixture %d: status=%d body=%s", j.idx, r.Status, string(r.Body))
					return
				}
				fx.AddUser(fixtures.User{
					Email:    email,
					Password: validBenchPassword,
					UserID:   resp.ID,
					Token:    resp.Token,
				})
			}
			errCh <- nil
		}()
	}
	var firstErr error
	for w := 0; w < setupConcurrency; w++ {
		if e := <-errCh; e != nil && firstErr == nil {
			firstErr = e
		}
	}
	return firstErr
}

func (s *LoginBurst) Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result {
	hist := metrics.New()
	ok := metrics.NewCounter()
	errs := metrics.NewCounter()

	users := fx.SnapshotUsers()
	if len(users) == 0 {
		return Result{Name: s.Name(), Errors: 1, Extra: map[string]any{"error": "no fixture users"}}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rngMu := make(chan struct{}, 1)
	rngMu <- struct{}{}

	body := func(workerID int) (bool, time.Duration) {
		<-rngMu
		idx := rng.Intn(len(users))
		rngMu <- struct{}{}

		u := users[idx]
		payload := map[string]string{"email": u.Email, "password": u.Password}
		start := time.Now()
		r, err := c.Post(ctx, "/api/v1/auth/login", payload, nil)
		lat := time.Since(start)
		if err != nil {
			return false, lat
		}
		if r.Status >= 200 && r.Status < 300 {
			return true, lat
		}
		return false, lat
	}

	start := time.Now()
	runWorkers(opts.Concurrency, opts.Duration, hist, ok, errs, body)
	dur := time.Since(start)

	extra := map[string]any{
		"fixture_users": len(users),
	}
	return finalize(s.Name(), hist, ok, errs, dur, extra)
}

func (s *LoginBurst) Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error {
	return nil
}
