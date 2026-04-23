package proxy

import (
	"net/http/httptest"
	"testing"
)

func TestRepro_TrailingSpaceInPath(t *testing.T) {
	// Rule with trailing space in path
	e := mustEngine(t, RuleSpec{Path: "/api/* ", Allow: "anonymous"})

	req := httptest.NewRequest("GET", "/api/foo", nil)
	d := e.Evaluate(req, Identity{})
	if !d.Allow {
		t.Errorf("Expected allow for '/api/foo' with rule '/api/* ', got deny: %s", d.Reason)
	}
}

func TestRepro_OverlappingRules(t *testing.T) {
	// Rule 1 matches path but fails requirement
	// Rule 2 matches path and would satisfy requirement
	// Standard policy: First Match Wins (Deny wins if first rule requires)
	e := mustEngine(t,
		RuleSpec{Path: "/dashboard", Require: "role:admin"},
		RuleSpec{Path: "/dashboard", Require: "authenticated"},
	)

	id := Identity{UserID: "u1"} // Authenticated but not admin
	req := httptest.NewRequest("GET", "/dashboard", nil)

	d := e.Evaluate(req, id)
	if d.Allow {
		t.Errorf("Expected DENY for authenticated user on /dashboard because Rule 1 wins (Standard First Match Wins policy)")
	}
	if d.Reason != "role \"admin\" required" {
		t.Errorf("Expected reason 'role \"admin\" required', got %q", d.Reason)
	}
}

func TestRepro_AppIDMismatch(t *testing.T) {
	// Engine contains rules for multiple apps
	e := mustEngine(t,
		RuleSpec{AppID: "app-a", Path: "/api/*", Require: "role:admin"},
		RuleSpec{AppID: "app-b", Path: "/api/*", Allow: "anonymous"},
	)

	// Request for App B
	req := httptest.NewRequest("GET", "/api/foo", nil)
	req.Header.Set("X-Shark-App-ID", "app-b")

	d := e.Evaluate(req, Identity{})
	if !d.Allow {
		t.Errorf("Request for App B should have allowed via app-b rule, but got deny: %s", d.Reason)
	}
	if d.MatchedRule == nil || d.MatchedRule.AppID != "app-b" {
		t.Errorf("Expected match with app-b, got %v", d.MatchedRule)
	}
}
