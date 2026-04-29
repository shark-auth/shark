package api_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// seedUnverifiedUser inserts a user with email_verified=0 so the verify
// endpoint has something to flip + a welcome email to fire.
func seedUnverifiedUser(t *testing.T, ts *testutil.TestServer, id, email string) *storage.User {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	u := &storage.User{
		ID:            id,
		Email:         email,
		EmailVerified: false,
		HashType:      "argon2id",
		Metadata:      "{}",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := ts.Store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

// countWelcomeMessages returns how many captured emails look like the
// welcome email for the given address. Welcome subject starts with
// "Welcome to " (both the seeded DB template and the hardcoded fallback
// share that prefix).
func countWelcomeMessages(ts *testutil.TestServer, to string) int {
	n := 0
	for _, m := range ts.EmailSender.MessagesTo(to) {
		if len(m.Subject) >= len("Welcome to ") && m.Subject[:len("Welcome to ")] == "Welcome to " {
			n++
		}
	}
	return n
}

// waitForWelcome polls the captured inbox for up to 2s waiting on the
// welcome email. The handler dispatches the send in a goroutine so the
// HTTP response returns before the message is captured â€” a short poll is
// cheaper and less flaky than a fixed sleep.
func waitForWelcome(t *testing.T, ts *testutil.TestServer, to string, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if countWelcomeMessages(ts, to) >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %d welcome email(s) to %s; got %d", want, to, countWelcomeMessages(ts, to))
}

// TestWelcomeEmail_FiredOnFirstVerification drives the verify endpoint end
// to end: send verification â†’ click token â†’ assert welcome email landed
// in the mock inbox + welcome_email_sent flipped to 1 in the DB.
func TestWelcomeEmail_FiredOnFirstVerification(t *testing.T) {
	ts := testutil.NewTestServer(t)

	userEmail := "welcome-first@example.com"
	u := seedUnverifiedUser(t, ts, "usr_welcome_first", userEmail)

	// Generate a verify token via the manager directly â€” mirrors what the
	// /verify/send handler does, without needing an authenticated session.
	if err := ts.APIServer.MagicLinkManager.SendEmailVerification(context.Background(), userEmail); err != nil {
		t.Fatalf("SendEmailVerification: %v", err)
	}

	// The verify email is the first captured message. Extract the token
	// from its HTML body (same pattern as magic link tests).
	verifyMsgs := ts.EmailSender.MessagesTo(userEmail)
	if len(verifyMsgs) != 1 {
		t.Fatalf("expected 1 verify email, got %d", len(verifyMsgs))
	}
	token := extractTokenFromEmail(t, verifyMsgs[0].HTML)

	// Hit the verify endpoint.
	resp := ts.Get("/api/v1/auth/email/verify?token=" + url.QueryEscape(token))
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("verify: expected 200, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Welcome email fires in a goroutine â€” poll.
	waitForWelcome(t, ts, userEmail, 1)

	// DB flag should now be 1 â€” a second MarkWelcomeEmailSent returns
	// sql.ErrNoRows (tested exhaustively in storage unit test).
	if err := ts.Store.MarkWelcomeEmailSent(context.Background(), u.ID); err == nil {
		t.Fatalf("expected welcome_email_sent already=1 after verify, but second Mark succeeded")
	}
}

// TestWelcomeEmail_NotFiredOnSecondVerification pre-sets welcome_email_sent=1
// so the UPDATE guard matches zero rows â€” handler must NOT dispatch a
// second welcome email even though the verify endpoint ran successfully.
func TestWelcomeEmail_NotFiredOnSecondVerification(t *testing.T) {
	ts := testutil.NewTestServer(t)

	userEmail := "welcome-second@example.com"
	u := seedUnverifiedUser(t, ts, "usr_welcome_second", userEmail)

	// Pre-flip the flag â€” simulates a prior verification that already
	// triggered the welcome email.
	if err := ts.Store.MarkWelcomeEmailSent(context.Background(), u.ID); err != nil {
		t.Fatalf("pre-seed MarkWelcomeEmailSent: %v", err)
	}

	if err := ts.APIServer.MagicLinkManager.SendEmailVerification(context.Background(), userEmail); err != nil {
		t.Fatalf("SendEmailVerification: %v", err)
	}

	verifyMsgs := ts.EmailSender.MessagesTo(userEmail)
	if len(verifyMsgs) != 1 {
		t.Fatalf("expected 1 verify email, got %d", len(verifyMsgs))
	}
	token := extractTokenFromEmail(t, verifyMsgs[0].HTML)

	resp := ts.Get("/api/v1/auth/email/verify?token=" + url.QueryEscape(token))
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("verify: expected 200, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Give the handler a chance to (incorrectly) dispatch â€” even if the
	// goroutine ran, the flag was already 1 so no welcome must land.
	time.Sleep(100 * time.Millisecond)

	if got := countWelcomeMessages(ts, userEmail); got != 0 {
		t.Fatalf("expected 0 welcome emails on re-verification, got %d", got)
	}
}
