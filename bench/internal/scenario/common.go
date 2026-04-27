package scenario

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/sharkauth/bench/internal/metrics"
)

// runID is unique per process to keep email addresses unique across runs.
var runID = func() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}()

// randID returns a short hex string suitable for unique suffixes.
func randID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// uniqueEmail returns a fresh email per call.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s-%s@bench.local", prefix, runID, randID())
}

// validBenchPassword satisfies common policies (≥8, mixed case, digit).
const validBenchPassword = "BenchP@ss123"

// finalize assembles a Result from a histogram + counters + duration.
func finalize(name string, hist *metrics.Histogram, ok, errs *metrics.Counter, dur time.Duration, extra map[string]any) Result {
	okN := ok.Load()
	throughput := 0.0
	if dur > 0 {
		throughput = float64(okN) / dur.Seconds()
	}
	return Result{
		Name:       name,
		OK:         okN,
		Errors:     errs.Load(),
		LatencyP50: hist.Quantile(0.50),
		LatencyP95: hist.Quantile(0.95),
		LatencyP99: hist.Quantile(0.99),
		Throughput: throughput,
		Extra:      extra,
	}
}

// runWorkers fans out N goroutines and runs body until ctx is done.
// body returns (ok bool, latency time.Duration). Records to hist + ok/errs.
func runWorkers(concurrency int, dur time.Duration, hist *metrics.Histogram, ok, errs *metrics.Counter, body func(workerID int) (bool, time.Duration)) {
	var wg sync.WaitGroup
	deadline := time.Now().Add(dur)
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			for time.Now().Before(deadline) {
				success, lat := body(id)
				hist.Record(int64(lat))
				if success {
					ok.Inc()
				} else {
					errs.Inc()
				}
			}
		}(i)
	}
	wg.Wait()
}
