package storage

import "time"

// ProxyRule is the DB-backed override row for the reverse proxy rule engine
// (Phase 6.6 / Wave D). YAML rules from sharkauth.yaml remain the bootstrap
// source of truth; rows here are layered on top so admins can author + edit
// rules from the dashboard without restarting the server.
//
// Pattern is a chi-style path pattern (e.g. /api/orgs/{id}, /v1/public/*).
// Methods is the empty slice for "any method", otherwise an ordered list of
// uppercased HTTP verbs. Exactly one of Require/Allow must be set; the
// engine compiles them via the existing proxy.parseRequirement plumbing.
type ProxyRule struct {
	ID        string    `json:"id"` // pxr_<hex>
	AppID     string    `json:"app_id"`
	Name      string    `json:"name"`
	Pattern   string    `json:"pattern"`
	Methods   []string  `json:"methods"`
	Require   string    `json:"require"`
	Allow     string    `json:"allow"`
	Scopes    []string  `json:"scopes"`
	// TierMatch, if non-empty, constrains the rule to callers whose
	// Identity.Tier equals this value. The engine treats a mismatch as a
	// paywall redirect (DecisionPaywallRedirect) rather than a generic
	// 403 so the proxy can route browsers to an upgrade page. Lane A
	// migration 00023 added the column; v1.5 wires it through the CRUD
	// + engine paths.
	TierMatch string `json:"tier_match"`
	// M2M, when true, requires the caller's identity to be agent-typed
	// (ActorType == ActorTypeAgent). Humans carrying a valid JWT are
	// denied with "rule requires agent (m2m)". Added in PROXYV1_5 §4.17
	// so operators can lock endpoints to service-to-service traffic.
	M2M       bool      `json:"m2m"`
	Enabled   bool      `json:"enabled"`
	Priority  int       `json:"priority"`   // higher = evaluated first
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
