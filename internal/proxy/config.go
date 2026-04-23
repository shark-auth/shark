// Package proxy implements SharkAuth's reverse proxy core. It forwards
// authenticated HTTP traffic to a configured upstream, injecting identity
// headers (X-User-*, X-Agent-*, X-Shark-*) derived from the request's
// authenticated context so upstream services can trust them without
// re-validating tokens.
//
// This package intentionally knows nothing about auth — it consumes an
// Identity value placed into the request context by middleware earlier in
// the chain. The rules engine, circuit breaker, and dashboard wiring live
// in sibling tasks (P2–P5 of Phase 6).
package proxy

import (
	"errors"
	"time"
)

// Config configures a ReverseProxy instance.
type Config struct {
	// Enabled is the master switch. When false, callers should not wire the
	// proxy into the router at all; New() still succeeds so tests can build
	// a disabled config without errors.
	Enabled bool

	// Upstream is the base URL of the backend service, e.g.
	// "http://localhost:3000". Scheme + host are required; path/query are
	// ignored (the proxy preserves the inbound request's path and query).
	Upstream string

	// Timeout is the per-request upstream timeout. Zero means use the
	// package default (30s). Applied as a ResponseHeaderTimeout on the
	// transport and as a context deadline on the outbound request.
	Timeout time.Duration

	// BufferSize is the response buffer size in bytes. Zero means use the
	// net/http/httputil default.
	BufferSize int

	// TrustedHeaders is an allowlist of client-supplied headers that
	// survive StripIdentityHeaders even when their name matches a stripped
	// prefix (X-User-*, X-Agent-*, X-Shark-*). Non-prefixed headers are
	// always preserved and do not need to be listed here. Intended as an
	// escape hatch for specific deployment quirks — leave empty by default.
	TrustedHeaders []string

	// StripIncoming controls whether inbound identity headers from the
	// client are stripped before the proxy injects its own. Default is
	// true and that is the only secure setting; exposed as a field so
	// tests can exercise the unsafe path.
	StripIncoming bool

	// Rules is the route-level authorization rule list, evaluated first
	// match wins. Empty list + a non-nil Engine means default-deny for
	// every request; this field is informational — the actual compiled
	// engine is passed to New() separately so the Config struct stays
	// YAML-serializable and free of precompiled state.
	Rules []RuleSpec
}

// DefaultTimeout is applied when Config.Timeout is zero.
const DefaultTimeout = 30 * time.Second

// Validate reports configuration errors. Called by New(); callers that
// build a Config by hand can also invoke it directly.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Upstream == "" {
		return errors.New("proxy: upstream is required when proxy is enabled")
	}
	return nil
}
