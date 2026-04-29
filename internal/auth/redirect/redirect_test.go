package redirect_test

import (
	"testing"

	"github.com/shark-auth/shark/internal/auth/redirect"
)

// app returns an Application with all three allowlists populated for use in tests.
func app(callbacks, logouts, origins []string) *redirect.Application {
	return &redirect.Application{
		AllowedCallbackURLs: callbacks,
		AllowedLogoutURLs:   logouts,
		AllowedOrigins:      origins,
	}
}

func TestValidate_ExactMatch(t *testing.T) {
	a := app([]string{"https://example.com/callback"}, nil, nil)

	cases := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"exact match", "https://example.com/callback", nil},
		{"different path", "https://example.com/other", redirect.ErrNotAllowed},
		{"different scheme", "http://example.com/callback", redirect.ErrNotAllowed},
		{"different host", "https://evil.com/callback", redirect.ErrNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != tc.wantErr {
				t.Errorf("Validate(%q) = %v; want %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidate_WildcardSubdomain(t *testing.T) {
	a := app([]string{"https://*.preview.vercel.app"}, nil, nil)

	cases := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"single subdomain matches", "https://abc.preview.vercel.app", nil},
		{"another single subdomain", "https://feature-123.preview.vercel.app", nil},
		{"no subdomain", "https://preview.vercel.app", redirect.ErrNotAllowed},
		{"two levels deep", "https://a.b.preview.vercel.app", redirect.ErrNotAllowed},
		{"wrong scheme", "http://abc.preview.vercel.app", redirect.ErrNotAllowed},
		{"unrelated domain", "https://abc.evil.com", redirect.ErrNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != tc.wantErr {
				t.Errorf("Validate(%q) = %v; want %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidate_WildcardNoPathInjection(t *testing.T) {
	// Wildcard patterns with paths should not leak path-based matching.
	a := app([]string{"https://*.example.com/safe"}, nil, nil)

	cases := []struct {
		name    string
		url     string
		wantErr error
	}{
		// The pattern normalised is "https://*.example.com/safe"; since the normalised
		// requested URL must match the normalised pattern exactly (after wildcard check),
		// only the subdomain+baseDomain check applies â€” path in pattern restricts match.
		{"correct path", "https://sub.example.com/safe", nil},
		{"different path", "https://sub.example.com/unsafe", redirect.ErrNotAllowed},
		// Nested subdomain must be rejected even if path matches.
		{"nested subdomain", "https://a.b.example.com/safe", redirect.ErrNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != tc.wantErr {
				t.Errorf("Validate(%q) = %v; want %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidate_LoopbackAnyPort(t *testing.T) {
	// RFC 8252 Â§8.3: loopback pattern must allow any port.
	a := app([]string{"http://127.0.0.1"}, nil, nil)

	cases := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"standard port", "http://127.0.0.1:8080/callback", nil},
		{"ephemeral port", "http://127.0.0.1:54321/cb", nil},
		{"no port", "http://127.0.0.1/callback", nil},
		{"wrong scheme", "https://127.0.0.1/callback", redirect.ErrNotAllowed},
		{"localhost not in list", "http://localhost:8080/callback", redirect.ErrNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != tc.wantErr {
				t.Errorf("Validate(%q) = %v; want %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidate_LoopbackProductionRejected(t *testing.T) {
	// When the allowlist has only production URLs, loopback must be rejected.
	a := app([]string{"https://app.example.com/callback"}, nil, nil)

	cases := []struct {
		name string
		url  string
	}{
		{"localhost port", "http://localhost:8080/callback"},
		{"127.0.0.1", "http://127.0.0.1/callback"},
		{"127.0.0.1 with port", "http://127.0.0.1:3000/callback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != redirect.ErrNotAllowed {
				t.Errorf("Validate(%q) = %v; want ErrNotAllowed", tc.url, err)
			}
		})
	}
}

func TestValidate_BadScheme(t *testing.T) {
	a := app([]string{"https://example.com/cb"}, nil, nil)

	cases := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"javascript scheme", "javascript:alert(1)", redirect.ErrNotAllowed},
		{"file scheme", "file:///etc/passwd", redirect.ErrNotAllowed},
		// Empty string has no scheme â†’ ErrInvalidURL.
		{"empty url", "", redirect.ErrInvalidURL},
		// Custom native-app schemes are syntactically valid and may be in allowlist.
		{"custom scheme not in list", "com.example.app://callback", redirect.ErrNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != tc.wantErr {
				t.Errorf("Validate(%q) = %v; want %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidate_Userinfo(t *testing.T) {
	a := app([]string{"https://example.com/cb"}, nil, nil)

	cases := []struct {
		name string
		url  string
	}{
		{"user and password", "https://user:pass@example.com/cb"},
		{"user only", "https://user@example.com/cb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != redirect.ErrNotAllowed {
				t.Errorf("Validate(%q) = %v; want ErrNotAllowed", tc.url, err)
			}
		})
	}
}

func TestValidate_Fragment(t *testing.T) {
	a := app([]string{"https://example.com/cb"}, nil, nil)

	err := redirect.Validate(a, redirect.KindCallback, "https://example.com/cb#fragment")
	if err != redirect.ErrNotAllowed {
		t.Errorf("Validate with fragment = %v; want ErrNotAllowed", err)
	}
}

func TestValidate_NormalizationLowercase(t *testing.T) {
	// Both pattern and requested URL must normalise to the same string.
	a := app([]string{"https://example.com/cb"}, nil, nil)

	cases := []struct {
		name    string
		url     string
		wantErr error
	}{
		{"uppercase scheme+host", "HTTPS://EXAMPLE.COM/cb", nil},
		{"mixed case", "Https://Example.Com/cb", nil},
		{"mixed case no match", "HTTPS://EVIL.COM/cb", redirect.ErrNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != tc.wantErr {
				t.Errorf("Validate(%q) = %v; want %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidate_TrailingSlash(t *testing.T) {
	// Pattern without trailing slash and URL with trailing slash (and vice-versa)
	// must both match, because normalize strips a trailing "/" when path == "/".
	cases := []struct {
		name     string
		pattern  string
		url      string
		wantErr  error
	}{
		{"pattern no slash, url with slash", "https://example.com", "https://example.com/", nil},
		{"pattern with slash, url no slash", "https://example.com/", "https://example.com", nil},
		{"both no slash", "https://example.com", "https://example.com", nil},
		{"both with slash", "https://example.com/", "https://example.com/", nil},
		// A real path must not be stripped.
		{"real path unchanged", "https://example.com/path", "https://example.com/path", nil},
		{"real path mismatch", "https://example.com/path", "https://example.com/other", redirect.ErrNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := app([]string{tc.pattern}, nil, nil)
			err := redirect.Validate(a, redirect.KindCallback, tc.url)
			if err != tc.wantErr {
				t.Errorf("Validate(%q, pattern=%q) = %v; want %v", tc.url, tc.pattern, err, tc.wantErr)
			}
		})
	}
}

func TestValidate_KindRouting(t *testing.T) {
	// Ensure the kind parameter correctly selects the allowlist.
	a := &redirect.Application{
		AllowedCallbackURLs: []string{"https://example.com/callback"},
		AllowedLogoutURLs:   []string{"https://example.com/logout"},
		AllowedOrigins:      []string{"https://example.com"},
	}

	if err := redirect.Validate(a, redirect.KindCallback, "https://example.com/callback"); err != nil {
		t.Errorf("KindCallback expected nil, got %v", err)
	}
	if err := redirect.Validate(a, redirect.KindLogout, "https://example.com/logout"); err != nil {
		t.Errorf("KindLogout expected nil, got %v", err)
	}
	if err := redirect.Validate(a, redirect.KindOrigin, "https://example.com"); err != nil {
		t.Errorf("KindOrigin expected nil, got %v", err)
	}

	// Cross-kind must fail.
	if err := redirect.Validate(a, redirect.KindLogout, "https://example.com/callback"); err != redirect.ErrNotAllowed {
		t.Errorf("KindLogout with callback URL = %v; want ErrNotAllowed", err)
	}
}
