// Package metrics provides simple histogram + counter primitives for the bench harness.
//
// We avoid the hdrhistogram-go dependency to keep the bench module dep-free.
// The implementation is a sorted-slice histogram with interpolated quantiles.
// Range: 1us..60s, but Record() simply stores the raw nanoseconds — no clamping.
package metrics

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Histogram is a thread-safe latency recorder.
// Quantile reads are O(N log N) on first call after a Record; cached afterwards.
type Histogram struct {
	mu     sync.Mutex
	values []int64 // nanoseconds
	sorted bool
	sum    int64
	max    int64
}

// New returns an empty Histogram.
func New() *Histogram {
	return &Histogram{values: make([]int64, 0, 4096)}
}

// Record stores a single latency observation in nanoseconds.
func (h *Histogram) Record(latencyNanos int64) {
	if latencyNanos < 0 {
		latencyNanos = 0
	}
	h.mu.Lock()
	h.values = append(h.values, latencyNanos)
	h.sorted = false
	h.sum += latencyNanos
	if latencyNanos > h.max {
		h.max = latencyNanos
	}
	h.mu.Unlock()
}

// Count returns the number of recorded samples.
func (h *Histogram) Count() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return int64(len(h.values))
}

// Mean returns the arithmetic mean latency.
func (h *Histogram) Mean() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.values) == 0 {
		return 0
	}
	return time.Duration(h.sum / int64(len(h.values)))
}

// Max returns the maximum recorded latency.
func (h *Histogram) Max() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	return time.Duration(h.max)
}

// Quantile returns the latency at quantile q in [0,1] using linear interpolation.
func (h *Histogram) Quantile(q float64) time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := len(h.values)
	if n == 0 {
		return 0
	}
	if !h.sorted {
		sort.Slice(h.values, func(i, j int) bool { return h.values[i] < h.values[j] })
		h.sorted = true
	}
	if q <= 0 {
		return time.Duration(h.values[0])
	}
	if q >= 1 {
		return time.Duration(h.values[n-1])
	}
	pos := q * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return time.Duration(h.values[lo])
	}
	frac := pos - float64(lo)
	val := float64(h.values[lo]) + frac*(float64(h.values[hi])-float64(h.values[lo]))
	return time.Duration(val)
}

// Counter is a simple atomic int64.
type Counter struct {
	v atomic.Int64
}

// NewCounter returns a zeroed Counter.
func NewCounter() *Counter { return &Counter{} }

// Inc adds 1.
func (c *Counter) Inc() { c.v.Add(1) }

// Add adds n.
func (c *Counter) Add(n int64) { c.v.Add(n) }

// Load returns the current value.
func (c *Counter) Load() int64 { return c.v.Load() }
