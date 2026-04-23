package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders returns middleware that sets OWASP-recommended security response headers.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("X-XSS-Protection", "0")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			// Phase Wave E: Allow framing for branding preview iframe.
			// We check the path directly to be more robust than query params.
			if strings.HasPrefix(r.URL.Path, "/hosted/") || r.URL.Query().Get("preview") == "true" {
				h.Set("X-Frame-Options", "SAMEORIGIN")
				h.Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline'; frame-ancestors 'self'")
			} else {
				h.Set("X-Frame-Options", "DENY")
				h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
			}

			// HSTS only on HTTPS
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
