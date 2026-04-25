// Package middleware provides HTTP middleware for the SharkAuth API server.
package middleware

import (
	"net/http"
	"sync/atomic"
)

// DrainFlag is a shared atomic bool that signals whether the server is in
// drain mode (i.e. a DB swap or reset is in progress).
// Handlers check IsDraining(); the swap/reset logic calls SetDraining().
type DrainFlag struct {
	v atomic.Bool
}

// SetDraining sets the drain flag to the given value.
func (d *DrainFlag) SetDraining(v bool) { d.v.Store(v) }

// IsDraining reports whether drain mode is active.
func (d *DrainFlag) IsDraining() bool { return d.v.Load() }

// Drain returns an HTTP middleware that returns 503 + Retry-After: 2
// while the drain flag is set. New requests are rejected immediately so
// in-flight requests can drain before the store is swapped.
func Drain(flag *DrainFlag) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if flag.IsDraining() {
				w.Header().Set("Retry-After", "2")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":"service_unavailable","message":"Server is reloading, please retry in a moment"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
