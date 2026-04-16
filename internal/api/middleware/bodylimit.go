package middleware

import (
	"net/http"
)

// MaxBodySize returns middleware that limits the size of request bodies.
// Requests exceeding maxBytes will receive a 413 Request Entity Too Large response.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
