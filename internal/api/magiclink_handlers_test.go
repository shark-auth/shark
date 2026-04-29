package api_test

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

// tokenRegex extracts a token from a magic link URL in the email body.
var tokenRegex = regexp.MustCompile(`[?&]token=([A-Za-z0-9_-]+)`)

// extractTokenFromEmail extracts the magic link token from a captured email's HTML body.
func extractTokenFromEmail(t *testing.T, html string) string {
	t.Helper()
	matches := tokenRegex.FindStringSubmatch(html)
	if len(matches) < 2 {
		t.Fatalf("could not extract token from email body:\n%s", html)
	}
	return matches[1]
}

func TestMagicLinkFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)

	email := "magic@example.com"

	// 1. Send magic link
	resp := ts.PostJSON("/api/v1/auth/magic-link/send", map[string]string{
		"email": email,
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for send, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 2. Verify email was captured
	if ts.EmailSender.MessageCount() != 1 {
		t.Fatalf("expected 1 email sent, got %d", ts.EmailSender.MessageCount())
	}
	msg := ts.EmailSender.LastMessage()
	if msg.To != email {
		t.Fatalf("expected email to %s, got %s", email, msg.To)
	}

	// 3. Extract token from email HTML
	token := extractTokenFromEmail(t, msg.HTML)

	// 4. Verify the magic link token
	resp = ts.Get("/api/v1/auth/magic-link/verify?token=" + url.QueryEscape(token))
	if resp.StatusCode != http.StatusFound {
		body := readBody(t, resp)
		t.Fatalf("expected 302 redirect for verify, got %d: %s", resp.StatusCode, body)
	}
	// Check redirect location
	location := resp.Header.Get("Location")
	if location != ts.Config.MagicLink.RedirectURL {
		t.Fatalf("expected redirect to %s, got %s", ts.Config.MagicLink.RedirectURL, location)
	}
	resp.Body.Close()

	// 5. Session should be active â€” GET /me should work
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /me after magic link verify, got %d: %s", resp.StatusCode, body)
	}
	var meResult map[string]interface{}
	ts.DecodeJSON(resp, &meResult)
	if meResult["email"] != email {
		t.Fatalf("expected email %s in /me, got %v", email, meResult["email"])
	}
	if meResult["emailVerified"] != true {
		t.Fatalf("expected emailVerified=true, got %v", meResult["emailVerified"])
	}
}

func TestMagicLinkOneTimeUse(t *testing.T) {
	ts := testutil.NewTestServer(t)

	email := "onetime@example.com"

	// Send magic link
	resp := ts.PostJSON("/api/v1/auth/magic-link/send", map[string]string{
		"email": email,
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for send, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Extract token
	msg := ts.EmailSender.LastMessage()
	token := extractTokenFromEmail(t, msg.HTML)

	// First verify â€” should succeed (302)
	resp = ts.Get("/api/v1/auth/magic-link/verify?token=" + url.QueryEscape(token))
	if resp.StatusCode != http.StatusFound {
		body := readBody(t, resp)
		t.Fatalf("expected 302 for first verify, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Second verify â€” same token should return 400
	resp = ts.Get("/api/v1/auth/magic-link/verify?token=" + url.QueryEscape(token))
	if resp.StatusCode != http.StatusBadRequest {
		body := readBody(t, resp)
		t.Fatalf("expected 400 for second verify, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	ts.DecodeJSON(resp, &result)
	if result["error"] != "token_used" {
		t.Fatalf("expected error 'token_used', got %q", result["error"])
	}
}

func TestMagicLinkCreatesNewUser(t *testing.T) {
	ts := testutil.NewTestServer(t)

	email := "newuser@example.com"

	// Verify user doesn't exist yet
	_, err := ts.Store.GetUserByEmail(context.Background(), email)
	if err == nil {
		t.Fatal("expected user to not exist before magic link flow")
	}

	// Send magic link
	resp := ts.PostJSON("/api/v1/auth/magic-link/send", map[string]string{
		"email": email,
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for send, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Extract token and verify
	msg := ts.EmailSender.LastMessage()
	token := extractTokenFromEmail(t, msg.HTML)

	resp = ts.Get("/api/v1/auth/magic-link/verify?token=" + url.QueryEscape(token))
	if resp.StatusCode != http.StatusFound {
		body := readBody(t, resp)
		t.Fatalf("expected 302 for verify, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Verify user was created with email_verified=true
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 for /me, got %d: %s", resp.StatusCode, body)
	}
	var meResult map[string]interface{}
	ts.DecodeJSON(resp, &meResult)
	if meResult["email"] != email {
		t.Fatalf("expected email %s, got %v", email, meResult["email"])
	}
	if meResult["emailVerified"] != true {
		t.Fatalf("expected emailVerified=true for new magic link user, got %v", meResult["emailVerified"])
	}
}

func TestMagicLinkInvalidToken(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Try to verify with a bogus token
	resp := ts.Get("/api/v1/auth/magic-link/verify?token=bogus-token-that-does-not-exist")
	if resp.StatusCode != http.StatusBadRequest {
		body := readBody(t, resp)
		t.Fatalf("expected 400 for invalid token, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	ts.DecodeJSON(resp, &result)
	if result["error"] != "invalid_token" {
		t.Fatalf("expected error 'invalid_token', got %q", result["error"])
	}
}

func TestMagicLinkMissingToken(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Try to verify without a token parameter
	resp := ts.Get("/api/v1/auth/magic-link/verify")
	if resp.StatusCode != http.StatusBadRequest {
		body := readBody(t, resp)
		t.Fatalf("expected 400 for missing token, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	ts.DecodeJSON(resp, &result)
	if result["error"] != "missing_token" {
		t.Fatalf("expected error 'missing_token', got %q", result["error"])
	}
}

func TestMagicLinkInvalidEmail(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Send with invalid email
	resp := ts.PostJSON("/api/v1/auth/magic-link/send", map[string]string{
		"email": "not-an-email",
	})
	if resp.StatusCode != http.StatusBadRequest {
		body := readBody(t, resp)
		t.Fatalf("expected 400 for invalid email, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	ts.DecodeJSON(resp, &result)
	if result["error"] != "invalid_email" {
		t.Fatalf("expected error 'invalid_email', got %q", result["error"])
	}
}

func TestMagicLinkAlwaysReturnsSuccess(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Sending a magic link to any email should always return 200
	// (even if the email doesn't exist) to avoid leaking info
	resp := ts.PostJSON("/api/v1/auth/magic-link/send", map[string]string{
		"email": "nobody@nonexistent.com",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200 (don't leak email existence), got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// An email should still have been sent (magic link creates user on verify, not on send)
	if ts.EmailSender.MessageCount() != 1 {
		t.Fatalf("expected 1 email sent even for unknown address, got %d", ts.EmailSender.MessageCount())
	}
}
