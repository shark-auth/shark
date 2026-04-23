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
	Enabled   bool      `json:"enabled"`
	Priority  int       `json:"priority"`   // higher = evaluated first
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
