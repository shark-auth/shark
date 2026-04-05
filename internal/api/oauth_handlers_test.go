package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/auth/providers"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

func TestOAuthGitHubCallback(t *testing.T) {
	// 1. Set up a fake GitHub API server that returns a token and user info
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
				"id":         12345,
				"login":      "sharkuser",
				"name":       "Shark User",
				"email":      "shark@example.com",
				"avatar_url": "https://github.com/images/sharkuser.png",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer fakeGitHub.Close()

	// 2. Create the test server
	ts := testutil.NewTestServer(t)

	// 3. Register a mock GitHub provider that points at the fake server
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

	// Wrap the GitHub provider to override GetUser with our mock user URL
	mockGH := &mockGitHubProvider{
		GitHub:  ghProvider,
		userURL: fakeGitHub.URL + "/api/user",
	}
	ts.APIServer.OAuthManager.RegisterProvider(mockGH)

	// 4. Simulate the callback with a matching state cookie
	state := "test-state-value"
	callbackURL := fmt.Sprintf("/api/v1/auth/oauth/github/callback?code=test-auth-code&state=%s", state)

	u, _ := url.Parse(ts.Server.URL)
	ts.Client.Jar.SetCookies(u, []*http.Cookie{
		{Name: "shark_oauth_state", Value: state, Path: "/"},
	})

	resp := ts.Get(callbackURL)
	defer resp.Body.Close()

	// 5. Assert the response
	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected status 200, got %d: %v", resp.StatusCode, errBody)
	}

	var userResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&userResp)

	if email, ok := userResp["email"].(string); !ok || email != "shark@example.com" {
		t.Errorf("expected email shark@example.com, got %v", userResp["email"])
	}

	// Verify a session cookie was set
	cookies := ts.Client.Jar.Cookies(u)
	var hasSessionCookie bool
	for _, c := range cookies {
		if c.Name == "shark_session" {
			hasSessionCookie = true
			break
		}
	}
	if !hasSessionCookie {
		t.Error("expected shark_session cookie to be set")
	}

	// 6. Verify user exists in the store
	user, err := ts.Store.GetUserByEmail(t.Context(), "shark@example.com")
	if err != nil {
		t.Fatalf("user not found in store: %v", err)
	}
	if user.Email != "shark@example.com" {
		t.Errorf("expected email shark@example.com, got %s", user.Email)
	}

	// 7. Verify OAuth account was linked
	accounts, err := ts.Store.GetOAuthAccountsByUserID(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("getting oauth accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 oauth account, got %d", len(accounts))
	}
	if accounts[0].Provider != "github" {
		t.Errorf("expected provider github, got %s", accounts[0].Provider)
	}
	if accounts[0].ProviderID != "12345" {
		t.Errorf("expected provider_id 12345, got %s", accounts[0].ProviderID)
	}

	// 8. Verify second callback reuses same user (not a duplicate)
	ts.Client.Jar.SetCookies(u, []*http.Cookie{
		{Name: "shark_oauth_state", Value: state, Path: "/"},
	})
	resp2 := ts.Get(callbackURL)
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second callback: expected status 200, got %d", resp2.StatusCode)
	}

	var userResp2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&userResp2)

	if userResp2["id"] != userResp["id"] {
		t.Errorf("second callback created a different user: %v != %v", userResp2["id"], userResp["id"])
	}
}

func TestOAuthStartRedirectsToProvider(t *testing.T) {
	ts := testutil.NewTestServer(t)

	ts.APIServer.OAuthManager.RegisterProvider(&staticProvider{
		name:    "testprov",
		authURL: "https://provider.example.com/auth",
	})

	resp := ts.Get("/api/v1/auth/oauth/testprov")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("expected Location header")
	}

	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parsing Location: %v", err)
	}
	if parsed.Host != "provider.example.com" {
		t.Errorf("expected redirect to provider.example.com, got %s", parsed.Host)
	}

	// Verify state cookie was set and matches query param
	u, _ := url.Parse(ts.Server.URL)
	cookies := ts.Client.Jar.Cookies(u)
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "shark_oauth_state" {
			stateCookie = c
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("expected shark_oauth_state cookie")
	}
	if stateCookie.Value == "" {
		t.Error("expected non-empty state value")
	}

	queryState := parsed.Query().Get("state")
	if queryState != stateCookie.Value {
		t.Errorf("state mismatch: cookie=%s, query=%s", stateCookie.Value, queryState)
	}
}

func TestOAuthCallbackStateMismatch(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ts.APIServer.OAuthManager.RegisterProvider(&staticProvider{name: "testprov"})

	u, _ := url.Parse(ts.Server.URL)
	ts.Client.Jar.SetCookies(u, []*http.Cookie{
		{Name: "shark_oauth_state", Value: "correct-state", Path: "/"},
	})

	resp := ts.Get("/api/v1/auth/oauth/testprov/callback?code=x&state=wrong-state")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for state mismatch, got %d", resp.StatusCode)
	}
}

func TestOAuthUnknownProvider(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.Get("/api/v1/auth/oauth/nonexistent")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown provider, got %d", resp.StatusCode)
	}
}

// --- helpers ---

// mockGitHubProvider wraps the real GitHub provider but overrides GetUser
// to use a custom user URL (pointing at our fake server).
type mockGitHubProvider struct {
	*providers.GitHub
	userURL string
}

func (m *mockGitHubProvider) Name() string { return "github" }

func (m *mockGitHubProvider) GetUser(ctx context.Context, token *oauth2.Token) (*auth.OAuthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get(m.userURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
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
		AvatarURL:  data.AvatarURL,
	}, nil
}

// staticProvider is a minimal OAuthProvider for testing route behavior.
type staticProvider struct {
	name    string
	authURL string
}

func (p *staticProvider) Name() string { return p.name }
func (p *staticProvider) AuthURL(state string) string {
	return p.authURL + "?state=" + state
}
func (p *staticProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return nil, fmt.Errorf("not implemented")
}
func (p *staticProvider) GetUser(ctx context.Context, token *oauth2.Token) (*auth.OAuthUserInfo, error) {
	return nil, fmt.Errorf("not implemented")
}
