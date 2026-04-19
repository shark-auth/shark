package proxy

import (
	"net/http"
	"testing"
	"time"
)

func TestStripIdentityHeaders_RemovesPrefixed(t *testing.T) {
	h := http.Header{}
	h.Set("X-User-ID", "attacker")
	h.Set("X-Agent-Name", "fake-bot")
	h.Set("X-Shark-Cache-Age", "999")
	h.Set("X-Request-ID", "req-123")
	h.Set("X-Real-IP", "10.0.0.1")
	h.Set("Authorization", "Bearer xyz")

	StripIdentityHeaders(h, nil)

	if got := h.Get("X-User-ID"); got != "" {
		t.Errorf("X-User-ID not stripped, got %q", got)
	}
	if got := h.Get("X-Agent-Name"); got != "" {
		t.Errorf("X-Agent-Name not stripped, got %q", got)
	}
	if got := h.Get("X-Shark-Cache-Age"); got != "" {
		t.Errorf("X-Shark-Cache-Age not stripped, got %q", got)
	}
	if got := h.Get("X-Request-ID"); got != "req-123" {
		t.Errorf("X-Request-ID should survive, got %q", got)
	}
	if got := h.Get("X-Real-IP"); got != "10.0.0.1" {
		t.Errorf("X-Real-IP should survive, got %q", got)
	}
	if got := h.Get("Authorization"); got != "Bearer xyz" {
		t.Errorf("Authorization should survive, got %q", got)
	}
}

func TestStripIdentityHeaders_CaseInsensitive(t *testing.T) {
	// Directly set a lowercase key, bypassing canonicalization, to
	// confirm the strip pass doesn't depend on Set()'s normalization.
	h := http.Header{}
	h["x-user-id"] = []string{"sneaky"}
	h["X-AGENT-NAME"] = []string{"also-sneaky"}
	// Use the canonical form for the survivor so h.Get finds it; the
	// point of this test is the strip pass, not http.Header semantics.
	h.Set("X-Request-Id", "legit")

	StripIdentityHeaders(h, nil)

	for _, k := range []string{"x-user-id", "X-user-id", "X-User-ID", "X-AGENT-NAME", "X-Agent-Name"} {
		if v := h.Get(k); v != "" {
			t.Errorf("expected %q to be stripped, got %q", k, v)
		}
		if _, ok := h[k]; ok {
			t.Errorf("raw map still contains %q", k)
		}
	}
	if got := h.Get("X-Request-Id"); got != "legit" {
		t.Errorf("X-Request-Id should survive case-insensitive strip, got %q", got)
	}
}

func TestStripIdentityHeaders_TrustedAllowlist(t *testing.T) {
	h := http.Header{}
	h.Set("X-Shark-Trace-ID", "trace-42")
	h.Set("X-User-ID", "attacker")

	StripIdentityHeaders(h, []string{"X-Shark-Trace-ID"})

	if got := h.Get("X-Shark-Trace-ID"); got != "trace-42" {
		t.Errorf("trusted header should survive, got %q", got)
	}
	if got := h.Get("X-User-ID"); got != "" {
		t.Errorf("non-trusted identity header should be stripped, got %q", got)
	}
}

func TestInjectIdentity_EmitsSetFields(t *testing.T) {
	h := http.Header{}
	id := Identity{
		UserID:     "user-1",
		UserEmail:  "alice@example.com",
		UserRoles:  []string{"admin", "user"},
		AgentID:    "agent-7",
		AgentName:  "claude",
		AuthMethod: "jwt",
		CacheAge:   5 * time.Second,
	}
	InjectIdentity(h, id)

	want := map[string]string{
		HeaderUserID:     "user-1",
		HeaderUserEmail:  "alice@example.com",
		HeaderUserRoles:  "admin,user",
		HeaderAgentID:    "agent-7",
		HeaderAgentName:  "claude",
		HeaderAuthMethod: "jwt",
		HeaderAuthMode:   "jwt",
		HeaderCacheAge:   "5",
	}
	for k, v := range want {
		if got := h.Get(k); got != v {
			t.Errorf("header %s: got %q, want %q", k, got, v)
		}
	}
}

func TestInjectIdentity_OmitsEmptyFields(t *testing.T) {
	h := http.Header{}
	// Pre-populate with values that should be cleared — Inject must
	// overwrite (not leave stale data behind) for defense in depth.
	h.Set(HeaderUserEmail, "stale@example.com")
	h.Set(HeaderUserRoles, "stale-role")

	InjectIdentity(h, Identity{UserID: "user-1", AuthMethod: "jwt"})

	if got := h.Get(HeaderUserID); got != "user-1" {
		t.Errorf("X-User-ID: got %q, want user-1", got)
	}
	if got := h.Get(HeaderAuthMethod); got != "jwt" {
		t.Errorf("X-Auth-Method: got %q, want jwt", got)
	}
	if got := h.Get(HeaderUserEmail); got != "" {
		t.Errorf("X-User-Email should be cleared, got %q", got)
	}
	if got := h.Get(HeaderUserRoles); got != "" {
		t.Errorf("X-User-Roles should be cleared, got %q", got)
	}
	if got := h.Get(HeaderAgentID); got != "" {
		t.Errorf("X-Agent-ID should be absent, got %q", got)
	}
	if got := h.Get(HeaderCacheAge); got != "" {
		t.Errorf("X-Shark-Cache-Age should be absent when CacheAge=0, got %q", got)
	}
}

func TestInjectIdentity_EncodesRolesAsCommaJoined(t *testing.T) {
	h := http.Header{}
	InjectIdentity(h, Identity{UserRoles: []string{"admin", "user", "ops"}})
	if got := h.Get(HeaderUserRoles); got != "admin,user,ops" {
		t.Errorf("X-User-Roles: got %q, want admin,user,ops", got)
	}
}

func TestInjectIdentity_EmitsCacheAgeOnlyIfNonZero(t *testing.T) {
	h := http.Header{}
	InjectIdentity(h, Identity{UserID: "u"})
	if got := h.Get(HeaderCacheAge); got != "" {
		t.Errorf("CacheAge=0 should omit header, got %q", got)
	}

	h2 := http.Header{}
	InjectIdentity(h2, Identity{UserID: "u", CacheAge: 12 * time.Second})
	if got := h2.Get(HeaderCacheAge); got != "12" {
		t.Errorf("CacheAge=12s should emit \"12\", got %q", got)
	}
}

func TestInjectIdentity_OverwritesClientSuppliedHeaders(t *testing.T) {
	// If a client somehow slipped an identity header past the strip pass
	// (e.g. StripIncoming=false), Inject must still overwrite it.
	h := http.Header{}
	h.Set(HeaderUserID, "attacker")
	h.Set(HeaderAuthMethod, "spoofed")

	InjectIdentity(h, Identity{UserID: "real-user", AuthMethod: "jwt"})

	if got := h.Get(HeaderUserID); got != "real-user" {
		t.Errorf("X-User-ID should be overwritten, got %q", got)
	}
	if got := h.Get(HeaderAuthMethod); got != "jwt" {
		t.Errorf("X-Auth-Method should be overwritten, got %q", got)
	}
}
