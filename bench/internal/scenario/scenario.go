// Package scenario defines the Scenario contract for the bench harness.
package scenario

import (
	"context"
	"time"

	"github.com/sharkauth/bench/internal/client"
	"github.com/sharkauth/bench/internal/fixtures"
)

// Opts is the runtime configuration passed into a scenario's Run.
type Opts struct {
	Concurrency int
	Duration    time.Duration
	DPoPMode    string
	AdminKey    string // bearer key for admin-scoped fixtures
}

// Result is the aggregated outcome of one scenario run.
type Result struct {
	Name       string
	OK         int64
	Errors     int64
	LatencyP50 time.Duration
	LatencyP95 time.Duration
	LatencyP99 time.Duration
	Throughput float64 // RPS = OK / Duration
	Extra      map[string]any
}

// Scenario is implemented by every bench scenario.
type Scenario interface {
	Name() string
	Setup(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) error
	Run(ctx context.Context, c *client.Client, fx *fixtures.Bundle, opts Opts) Result
	Teardown(ctx context.Context, c *client.Client, fx *fixtures.Bundle) error
}
