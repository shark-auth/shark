package audit

import (
	"net"
	"net/http"
	"strings"
)

// ExtractRequestInfo extracts the client IP and user agent from an HTTP request.
// It checks X-Forwarded-For and X-Real-Ip headers before falling back to RemoteAddr.
func ExtractRequestInfo(r *http.Request) (ip, userAgent string) {
	userAgent = r.UserAgent()
	ip = extractIP(r)
	return ip, userAgent
}

// extractIP returns the client IP from the request, checking proxy headers first.
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For (may contain comma-separated list)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}

	// Check X-Real-Ip
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (strip port)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
