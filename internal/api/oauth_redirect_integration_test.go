//go:build integration

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/auth/providers"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// setupOAuthRedirectTest creates a test server with a mock GitHub provider
// pointing at a fake GitHub API, and returns the test server along with a
// function that performs a full OAuth callback for a given redirect_uri parameter.
func setupOAuthRedirectTest(t *testing.T) (ts *testutil.TestServer, doCallback func(redirectURI string) *http.Response) {
	t.Helper()

	fakeGitHub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "fake-access-token",
				"token_type":   "bearer",
				"scope":        "user:email",
			})
		case "/api/user":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         99999,
				"login":      "redirecttest",
				"name":       "Redirect Test",
				"email":      "redirect-test@example.com",
				"avatar_url": "",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(fakeGitHub.Close)

	ts = testutil.NewTestServer(t)

	ghProvider := providers.NewGitHubWithConfig(oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  fakeGitHub.URL + "/login/oauth/authorize",
			TokenURL: fakeGitHub.URL + "/login/oauth/access_token",
		},
		RedirectURL: ts.Server.URL + "/api/v1/auth/oauth/github/callback",
		Scopes:      []string{"user:email"},
	})
	mockGH := &mockGHProvider{
		GitHub:  ghProvider,
		userURL: fakeGitHub.URL + "/api/user",
	}
	ts.APIServer.OAuthManager.RegisterProvider(mockGH)

	doCallback = func(redirectURI string) *http.Response {
		state := fmt.Sprintf("test-state-%d", time.Now().UnixNano())
		cbURL := fmt.Sprintf("/api/v1/auth/oauth/github/callback?code=test-code&state=%s", state)
		if redirectURI != "" {
			cbURL += "&redirect_uri=" + url.QueryEscape(redirectURI)
		}

		u, _ := url.Parse(ts.Server.URL)
		ts.Client.Jar.SetCookies(u, []*http.Cookie{
			{Name: "shark_oauth_state", Value: state, Path: "/"},
		})

		return ts.Get(cbURL)
	}

	return ts, doCallback
}

// mockGHProvider is a minimal mock that wraps the real GitHub provider.
type mockGHProvider struct {
	*providers.GitHub
	userURL string
}

func (m *mockGHProvider) Name() string { return "github" }

func (m *mockGHProvider) GetUser(ctx context.Context, token *oauth2.Token) (*auth.OAuthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get(m.userURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	name := data.Name
	if name == "" {
		name = data.Login
	}
	return &auth.OAuthUserInfo{
		ProviderID: fmt.Sprintf("%d", data.ID),
		Email:      data.Email,
		Name:       name,
	}, nil
}

// TestOAuthCallback_RedirectInAllowlist_Succeeds verifies that a redirect_uri
// in the default app's allowlist results in a 302 redirect.
func TestOAuthCallback_RedirectInAllowlist_Succeeds(t *testing.T) {
	ts, doCallback := setupOAuthRedirectTest(t)

	// Add the target redirect URI to the default app's allowlist.
	allowedURI := "https://myapp.example.com/auth/callback"
	ctx := context.Background()
	defaultApp, err := ts.Store.GetDefaultApplication(ctx)
	if err != nil {
		t.Fatalf("get default app: %v", err)
	}
	defaultApp.AllowedCallbackURLs = append(defaultApp.AllowedCallbackURLs, allowedURI)
	if err := ts.Store.UpdateApplication(ctx, defaultApp); err != nil {
		t.Fatalf("update default app: %v", err)
	}

	resp := doCallback(allowedURI)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 302, got %d: %s", resp.StatusCode, string(body))
	}
	loc := resp.Header.Get("Location")
	if loc != allowedURI {
		t.Errorf("expected redirect to %s, got %s", allowedURI, loc)
	}
}

// TestOAuthCallback_RedirectNotAllowed_400 verifies that a redirect_uri NOT
// in the default app's allowlist results in a 400 (not a redirect to the bad URI).
func TestOAuthCallback_RedirectNotAllowed_400(t *testing.T) {
	_, doCallback := setupOAuthRedirectTest(t)

	badURI := "https://evil.attacker.com/steal-token"
	resp := doCallback(badURI)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 for disallowed redirect_uri, got %d: %s", resp.StatusCode, string(body))
	}

	// The response body must mention "not allowed" (plain-text error, no redirect).
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "not allowed") {
		t.Errorf("expected 'not allowed' in response body, got: %s", string(body))
	}

	// Verify Location header is NOT set (we must not redirect to the bad URI).
	loc := resp.Header.Get("Location")
	if loc != "" {
		t.Errorf("should not redirect to bad URI; got Location: %s", loc)
	}
}
