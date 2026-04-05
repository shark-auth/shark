package middleware

import (
	"crypto/subtle"
	"net/http"
)

// AdminAPIKey returns a middleware that checks the X-Admin-Key header against
// the configured admin API key. Requests without a valid key receive 401.
// If the configured key is empty, all requests are allowed (development mode).
func AdminAPIKey(configuredKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no admin key is configured, allow all requests (dev mode)
			if configuredKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			providedKey := r.Header.Get("X-Admin-Key")
			if providedKey == "" {
				http.Error(w, `{"error":"unauthorized","message":"Missing X-Admin-Key header"}`, http.StatusUnauthorized)
				return
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(configuredKey)) != 1 {
				http.Error(w, `{"error":"unauthorized","message":"Invalid admin API key"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
