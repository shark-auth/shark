package email_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shark-auth/shark/internal/email"
)

// mockRedirectStore implements email.RedirectStore for testing.
type mockRedirectStore struct {
	raw string
	err error
}

func (m *mockRedirectStore) GetSystemConfig(_ context.Context) (string, error) {
	return m.raw, m.err
}

func makeConfig(t *testing.T, emailCfg map[string]string) string {
	t.Helper()
	emailBytes, err := json.Marshal(emailCfg)
	if err != nil {
		t.Fatalf("marshal email cfg: %v", err)
	}
	wrapper := map[string]json.RawMessage{
		"email_config": json.RawMessage(emailBytes),
	}
	b, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("marshal wrapper: %v", err)
	}
	return string(b)
}

func TestGetRedirectURL_Invite_ConfigSet(t *testing.T) {
	want := "https://myapp.com/accept"
	store := &mockRedirectStore{
		raw: makeConfig(t, map[string]string{
			"invite_redirect_url": want,
		}),
	}
	got, configured, err := email.GetRedirectURL(context.Background(), store, "invite", "https://fallback.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !configured {
		t.Error("expected configured=true, got false")
	}
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestGetRedirectURL_Invite_Unset_ReturnsFallback(t *testing.T) {
	fallback := "https://fallback.example.com"
	store := &mockRedirectStore{
		raw: makeConfig(t, map[string]string{
			// invite_redirect_url intentionally absent
			"verify_redirect_url": "https://myapp.com/verify",
		}),
	}
	got, configured, err := email.GetRedirectURL(context.Background(), store, "invite", fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if configured {
		t.Error("expected configured=false, got true")
	}
	if got != fallback {
		t.Errorf("expected fallback %q, got %q", fallback, got)
	}
}

func TestGetRedirectURL_Invite_EmptyConfig_ReturnsFallback(t *testing.T) {
	fallback := "https://fallback.example.com"
	store := &mockRedirectStore{raw: ""}
	got, configured, err := email.GetRedirectURL(context.Background(), store, "invite", fallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if configured {
		t.Error("expected configured=false, got true")
	}
	if got != fallback {
		t.Errorf("expected %q, got %q", fallback, got)
	}
}

func TestGetRedirectURL_Verify_ConfigSet(t *testing.T) {
	want := "https://myapp.com/verified"
	store := &mockRedirectStore{
		raw: makeConfig(t, map[string]string{
			"verify_redirect_url": want,
		}),
	}
	got, configured, err := email.GetRedirectURL(context.Background(), store, "verify", "https://fallback.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !configured {
		t.Error("expected configured=true, got false")
	}
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestGetRedirectURL_UnknownKind_ReturnsError(t *testing.T) {
	// Must provide a non-empty config that reaches the switch statement.
	store := &mockRedirectStore{
		raw: makeConfig(t, map[string]string{
			"verify_redirect_url": "https://app.com/verify",
		}),
	}
	_, _, err := email.GetRedirectURL(context.Background(), store, "bogus", "https://fallback.example.com")
	if err == nil {
		t.Error("expected error for unknown kind, got nil")
	}
}
