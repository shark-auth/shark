package api_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

func TestSignupLoginLogoutFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// 1. Signup -> 201, session cookie set
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "test@example.com",
		"password": "securepassword123",
		"name":     "Test User",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	var signupResult map[string]interface{}
	ts.DecodeJSON(resp, &signupResult)
	if signupResult["id"] == nil || signupResult["id"] == "" {
		t.Fatal("expected user ID in signup response")
	}
	if signupResult["email"] != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %v", signupResult["email"])
	}

	// 2. GET /me -> 200, correct user data
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /me, got %d: %s", resp.StatusCode, body)
	}
	var meResult map[string]interface{}
	ts.DecodeJSON(resp, &meResult)
	if meResult["email"] != "test@example.com" {
		t.Fatalf("expected email test@example.com in /me, got %v", meResult["email"])
	}
	if meResult["name"] != "Test User" {
		t.Fatalf("expected name 'Test User' in /me, got %v", meResult["name"])
	}

	// 3. POST /logout -> 200
	resp = ts.PostJSON("/api/v1/auth/logout", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /logout, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 4. GET /me -> 401
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusUnauthorized {
		body := readBody(t, resp)
		t.Fatalf("expected 401 for /me after logout, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 5. POST /login -> 200, new session cookie
	resp = ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email":    "test@example.com",
		"password": "securepassword123",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /login, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 6. GET /me -> 200 again
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /me after login, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

func TestSignupDuplicateEmail(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Signup -> 201
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "dupe@example.com",
		"password": "securepassword123",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Signup same email -> 409
	resp = ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "dupe@example.com",
		"password": "anotherpassword123",
	})
	if resp.StatusCode != http.StatusConflict {
		body := readBody(t, resp)
		t.Fatalf("expected 409, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

func TestLoginWrongPassword(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Signup -> 201
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "wrongpw@example.com",
		"password": "correctpassword123",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Logout first so we start fresh
	ts.PostJSON("/api/v1/auth/logout", nil).Body.Close()

	// Login wrong password -> 401
	resp = ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email":    "wrongpw@example.com",
		"password": "wrongpassword",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		body := readBody(t, resp)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}

func TestLoginNonexistentEmail(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Login unknown email -> 401 (same error as wrong password)
	resp := ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "somepassword123",
	})
	if resp.StatusCode != http.StatusUnauthorized {
		body := readBody(t, resp)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	ts.DecodeJSON(resp, &result)
	if result["error"] != "invalid_credentials" {
		t.Fatalf("expected error 'invalid_credentials', got %q", result["error"])
	}
}

func TestSessionRotation(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Signup (get session1)
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "rotation@example.com",
		"password": "securepassword123",
	})
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Get cookies after signup
	cookies1 := ts.Client.Jar.Cookies(parseURL(t, ts.Server.URL))
	session1 := findCookie(cookies1, "shark_session")
	if session1 == "" {
		t.Fatal("no session cookie after signup")
	}

	// Logout
	resp = ts.PostJSON("/api/v1/auth/logout", nil)
	resp.Body.Close()

	// Login (get session2)
	resp = ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email":    "rotation@example.com",
		"password": "securepassword123",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Get cookies after login
	cookies2 := ts.Client.Jar.Cookies(parseURL(t, ts.Server.URL))
	session2 := findCookie(cookies2, "shark_session")
	if session2 == "" {
		t.Fatal("no session cookie after login")
	}

	if session1 == session2 {
		t.Fatal("session cookie should be different after logout+login")
	}
}

// --- helpers ---

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return string(buf[:n])
}

func findCookie(cookies []*http.Cookie, name string) string {
	for _, c := range cookies {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

func parseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parsing URL: %v", err)
	}
	return u
}
