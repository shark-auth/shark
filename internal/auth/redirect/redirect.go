// Package redirect implements OAuth 2.1 redirect URI validation with support for
// exact match, wildcard subdomain, and loopback (RFC 8252 §8.3) patterns.
//
// It is a pure package — no dependency on storage, config, or net/http.
package redirect

import (
	"errors"
	"net/url"
	"strings"
)

var (
	// ErrNotAllowed is returned when the requested URL is not in the allowlist.
	ErrNotAllowed = errors.New("redirect URL not in allowlist")
	// ErrInvalidURL is returned when the requested URL cannot be parsed or has no scheme.
	ErrInvalidURL = errors.New("redirect URL not parseable")
)

// Kind selects which allowlist to validate against.
type Kind int

const (
	KindCallback Kind = iota // validate against AllowedCallbackURLs
	KindLogout               // validate against AllowedLogoutURLs
	KindOrigin               // validate against AllowedOrigins
)

// Application carries the allowlists from a registered application.
// This struct is intentionally separate from storage.Application to avoid
// import cycles; callers populate it from their preferred source.
type Application struct {
	AllowedCallbackURLs []string
	AllowedLogoutURLs   []string
	AllowedOrigins      []string
}

// Validate checks whether requestedURL is permitted for the given kind against app's allowlist.
//
// Rules (applied in order):
//  1. url.Parse(requestedURL) → ErrInvalidURL on error or empty scheme.
//  2. Scheme must not contain whitespace; empty scheme → ErrInvalidURL.
//  3. Userinfo present → ErrNotAllowed (OAuth 2.1 §3.1.2).
//  4. Fragment present → ErrNotAllowed (OAuth 2.1 §3.1.2).
//  5. Select allowlist by kind.
//  6. Normalise via normalize(u): lowercase scheme+host, strip default port (:80/:443),
//     strip trailing "/" when path is exactly "/".
//  7. Iterate allowlist with normalisation; first match wins:
//     - Exact match → allow.
//     - Wildcard: pattern starts with "https://*." → one-label subdomain check.
//     - Loopback: pattern is "http://127.0.0.1" or "http://localhost" → any port allowed.
//  8. No match → ErrNotAllowed.
func Validate(app *Application, kind Kind, requestedURL string) error {
	// Step 1 — parse.
	u, err := url.Parse(requestedURL)
	if err != nil || u.Scheme == "" {
		return ErrInvalidURL
	}

	// Step 2 — scheme sanity (no whitespace; catches "java script:" etc.).
	if strings.ContainsAny(u.Scheme, " \t\n\r") {
		return ErrInvalidURL
	}

	// Step 3 — userinfo banned.
	if u.User != nil {
		return ErrNotAllowed
	}

	// Step 4 — fragment banned (OAuth 2.1 §3.1.2).
	if u.Fragment != "" {
		return ErrNotAllowed
	}

	// Step 5 — select allowlist.
	var list []string
	switch kind {
	case KindCallback:
		list = app.AllowedCallbackURLs
	case KindLogout:
		list = app.AllowedLogoutURLs
	case KindOrigin:
		list = app.AllowedOrigins
	}

	// Step 6/7 — normalise and match.
	normReq := normalize(u)
	reqHost := stripPort(strings.ToLower(u.Host))

	for _, pattern := range list {
		pp, err := url.Parse(pattern)
		if err != nil || pp.Scheme == "" {
			continue // skip malformed patterns
		}
		normPat := normalize(pp)

		// Exact match.
		if normReq == normPat {
			return nil
		}

		// Wildcard subdomain: pattern must start with "https://*."
		if strings.HasPrefix(normPat, "https://*.") {
			// Extract base domain and path from the pattern (after "https://*.").
			rest := strings.TrimPrefix(normPat, "https://")
			rest = strings.TrimPrefix(rest, "*.")
			// Split host and path.
			patBaseDomain := rest
			patPath := ""
			if idx := strings.IndexByte(rest, '/'); idx != -1 {
				patBaseDomain = rest[:idx]
				patPath = rest[idx:]
			}
			if u.Scheme == "https" && matchWildcardSubdomain(reqHost, patBaseDomain) {
				// Path must match exactly (no path wildcard support).
				reqPath := u.Path
				if reqPath == "/" {
					reqPath = ""
				}
				if reqPath == patPath {
					return nil
				}
			}
			continue
		}

		// Loopback: RFC 8252 §8.3 — any port is permitted for loopback patterns.
		if normPat == "http://127.0.0.1" || normPat == "http://localhost" {
			patHost := strings.TrimPrefix(normPat, "http://")
			if u.Scheme == "http" && reqHost == patHost {
				return nil
			}
			continue
		}
	}

	return ErrNotAllowed
}

// normalize returns a canonical string for a parsed URL:
//   - scheme and host lowercased
//   - default port stripped (:80 for http, :443 for https)
//   - trailing "/" stripped when path is exactly "/"
func normalize(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)

	// Strip default ports.
	switch scheme {
	case "http":
		host = strings.TrimSuffix(host, ":80")
	case "https":
		host = strings.TrimSuffix(host, ":443")
	}

	path := u.Path
	if path == "/" {
		path = ""
	}

	// Reconstruct minimal form: scheme://host[path][?query]
	result := scheme + "://" + host + path
	if u.RawQuery != "" {
		result += "?" + u.RawQuery
	}
	return result
}

// stripPort removes the ":port" suffix from a host string.
func stripPort(host string) string {
	if idx := strings.LastIndexByte(host, ':'); idx != -1 {
		// Make sure we're not stripping an IPv6 address literal.
		if !strings.Contains(host, "]") || idx > strings.LastIndexByte(host, ']') {
			return host[:idx]
		}
	}
	return host
}

// matchWildcardSubdomain returns true if host is exactly one subdomain label
// immediately above baseDomain, with no nested dots in the label.
//
// e.g. matchWildcardSubdomain("abc.preview.vercel.app", "preview.vercel.app") == true
//
//	matchWildcardSubdomain("a.b.preview.vercel.app", "preview.vercel.app") == false
func matchWildcardSubdomain(host, baseDomain string) bool {
	suffix := "." + baseDomain
	if !strings.HasSuffix(host, suffix) {
		return false
	}
	label := strings.TrimSuffix(host, suffix)
	// Label must be non-empty and must not contain a dot (no nested wildcards).
	return label != "" && !strings.Contains(label, ".")
}
