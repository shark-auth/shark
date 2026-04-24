package identity

import (
	"context"
	"testing"
	"time"
)

// TestAuthMethodConstants pins the wire values of every AuthMethod. These
// strings are serialized into audit logs and upstream request headers, so
// changing them is a breaking change — the test exists to force a
// deliberate review when someone edits the constant block.
func TestAuthMethodConstants(t *testing.T) {
	cases := []struct {
		got  AuthMethod
		want string
	}{
		{AuthMethodCookie, "cookie"},
		{AuthMethodJWT, "jwt"},
		{AuthMethodAPIKey, "apikey"},
		{AuthMethodDPoP, "dpop"},
		{AuthMethodAnonymous, "anonymous"},
	}
	for _, c := range cases {
		if string(c.got) != c.want {
			t.Errorf("AuthMethod %q, want %q", string(c.got), c.want)
		}
	}
}

// TestActorTypeConstants pins the two actor-type values.
func TestActorTypeConstants(t *testing.T) {
	if string(ActorTypeHuman) != "human" {
		t.Errorf("ActorTypeHuman = %q, want %q", ActorTypeHuman, "human")
	}
	if string(ActorTypeAgent) != "agent" {
		t.Errorf("ActorTypeAgent = %q, want %q", ActorTypeAgent, "agent")
	}
}

// TestWithFromContext round-trips an Identity through the canonical
// context helpers. A present Identity must come back byte-equal; an
// absent one must return ok=false.
func TestWithFromContext(t *testing.T) {
	want := Identity{
		UserID:     "usr_1",
		UserEmail:  "a@b.c",
		Tier:       "pro",
		Roles:      []string{"admin", "member"},
		Scopes:     []string{"openid", "profile"},
		AuthMethod: AuthMethodJWT,
		ActorType:  ActorTypeHuman,
		SessionID:  "sess_x",
		MFAPassed:  true,
		CacheAge:   5 * time.Second,
	}
	ctx := WithIdentity(context.Background(), want)
	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext returned ok=false on populated ctx")
	}
	if got.UserID != want.UserID || got.Tier != want.Tier || got.AuthMethod != want.AuthMethod {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, want)
	}
	if len(got.Roles) != 2 || got.Roles[0] != "admin" {
		t.Errorf("Roles not preserved: %v", got.Roles)
	}
}

// TestFromContext_Absent verifies that an unannotated context reports
// "no identity" via ok=false rather than returning a misleading zero
// value indistinguishable from an empty-principal request.
func TestFromContext_Absent(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Error("FromContext returned ok=true on bare ctx")
	}
}

// TestFromContext_WrongType stashes a non-Identity under a colliding key
// (via context.WithValue through another package would, but the unexported
// key prevents external callers). Covered here for completeness by
// ensuring a zero-derived Identity in a parent chain is ignored.
func TestFromContext_WrongType(t *testing.T) {
	type other struct{}
	ctx := context.WithValue(context.Background(), other{}, "noise")
	if _, ok := FromContext(ctx); ok {
		t.Error("FromContext must only fire for values stored under its own key")
	}
}

// TestIsAnonymous — a request is anonymous when no principal fields are
// populated, regardless of AuthMethod (a stale "anonymous" string on an
// otherwise-empty identity still reports anonymous).
func TestIsAnonymous(t *testing.T) {
	cases := []struct {
		name string
		id   Identity
		want bool
	}{
		{"empty", Identity{}, true},
		{"user only", Identity{UserID: "u"}, false},
		{"agent only", Identity{AgentID: "a"}, false},
		{"both", Identity{UserID: "u", AgentID: "a"}, false},
		{"labelled anon", Identity{AuthMethod: AuthMethodAnonymous}, true},
	}
	for _, c := range cases {
		if got := c.id.IsAnonymous(); got != c.want {
			t.Errorf("%s: IsAnonymous = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestHasRole covers hit, miss, and empty-slice cases.
func TestHasRole(t *testing.T) {
	id := Identity{Roles: []string{"admin", "member"}}
	if !id.HasRole("admin") {
		t.Error("admin should match")
	}
	if id.HasRole("owner") {
		t.Error("owner should NOT match")
	}
	empty := Identity{}
	if empty.HasRole("admin") {
		t.Error("empty roles should never match")
	}
}

// TestHasScope covers hit + miss.
func TestHasScope(t *testing.T) {
	id := Identity{Scopes: []string{"openid", "profile", "webhooks:write"}}
	if !id.HasScope("webhooks:write") {
		t.Error("webhooks:write should match")
	}
	if id.HasScope("admin") {
		t.Error("admin should NOT match")
	}
}
