package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerCSPAllowsGitHubReleaseProbe(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "connect-src 'self' https://api.github.com") {
		t.Fatalf("CSP does not allow GitHub release probe: %q", csp)
	}
}
