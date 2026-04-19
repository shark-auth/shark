package proxy

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Identity is the resolved identity for an inbound request. The auth
// middleware places it into the request context via WithIdentity before
// ReverseProxy serves the request. Zero values for individual fields are
// valid — InjectIdentity simply omits the corresponding header.
type Identity struct {
	UserID     string
	UserEmail  string
	UserRoles  []string
	AgentID    string
	AgentName  string
	AuthMethod string        // "jwt" | "session-live" | "session-cached" | "anonymous"
	CacheAge   time.Duration // 0 if live; >0 if served from circuit-breaker cache
}

// Canonical header names injected by InjectIdentity and recognized by
// StripIdentityHeaders. Upstream services should trust these as the
// source of truth for the request's identity.
const (
	HeaderUserID     = "X-User-ID"
	HeaderUserEmail  = "X-User-Email"
	HeaderUserRoles  = "X-User-Roles"
	HeaderAgentID    = "X-Agent-ID"
	HeaderAgentName  = "X-Agent-Name"
	HeaderAuthMethod = "X-Auth-Method"
	HeaderCacheAge   = "X-Shark-Cache-Age"
	HeaderAuthMode   = "X-Shark-Auth-Mode"
)

// strippedPrefixes is the set of header name prefixes that
// StripIdentityHeaders removes from inbound requests. Upper-case because
// http.Header stores keys in canonical (textproto) form.
var strippedPrefixes = []string{"X-User-", "X-Agent-", "X-Shark-"}

// hasStrippedPrefix reports whether key begins with one of the reserved
// prefixes. Case-insensitive: http.Header canonicalizes keys on Set/Add,
// but raw maps and direct assignment can bypass canonicalization, so we
// compare against an upper-cased copy defensively.
func hasStrippedPrefix(key string) bool {
	upper := strings.ToUpper(key)
	for _, p := range strippedPrefixes {
		if strings.HasPrefix(upper, strings.ToUpper(p)) {
			return true
		}
	}
	return false
}

// StripIdentityHeaders removes any X-User-*, X-Agent-*, or X-Shark-*
// headers from h to prevent upstream-header spoofing by clients. Headers
// whose canonical name appears in trusted are preserved — this is an
// escape hatch for unusual deployments and should be used sparingly.
//
// Comparison is case-insensitive for both the prefix match and the
// trusted allowlist, mirroring http.Header's own case semantics.
func StripIdentityHeaders(h http.Header, trusted []string) {
	if len(h) == 0 {
		return
	}
	trustedSet := make(map[string]struct{}, len(trusted))
	for _, t := range trusted {
		trustedSet[http.CanonicalHeaderKey(t)] = struct{}{}
	}
	// Collect first to avoid mutating the map during range. Both the
	// canonical and raw keys are removed so headers that bypassed
	// canonicalization (e.g. by directly assigning to the underlying
	// map) are still stripped.
	var toDelete []string
	for k := range h {
		canon := http.CanonicalHeaderKey(k)
		if !hasStrippedPrefix(canon) {
			continue
		}
		if _, ok := trustedSet[canon]; ok {
			continue
		}
		toDelete = append(toDelete, k)
	}
	for _, k := range toDelete {
		delete(h, k)
		h.Del(k) // also clears the canonical form if it differs
	}
}

// InjectIdentity writes id's fields onto h as the canonical identity
// headers. Empty fields are omitted so upstream handlers can distinguish
// "not set" from "set to empty string". Existing values with the same
// canonical name are overwritten to prevent leakage through any header
// the strip pass might have missed.
func InjectIdentity(h http.Header, id Identity) {
	setOrDelete(h, HeaderUserID, id.UserID)
	setOrDelete(h, HeaderUserEmail, id.UserEmail)
	if len(id.UserRoles) > 0 {
		h.Set(HeaderUserRoles, strings.Join(id.UserRoles, ","))
	} else {
		h.Del(HeaderUserRoles)
	}
	setOrDelete(h, HeaderAgentID, id.AgentID)
	setOrDelete(h, HeaderAgentName, id.AgentName)
	setOrDelete(h, HeaderAuthMethod, id.AuthMethod)
	// X-Shark-Auth-Mode mirrors X-Auth-Method for upstream clarity; kept
	// as a distinct header so the rules engine can tag requests without
	// colliding with consumer-facing auth-method semantics.
	setOrDelete(h, HeaderAuthMode, id.AuthMethod)

	if id.CacheAge > 0 {
		h.Set(HeaderCacheAge, fmt.Sprintf("%d", int(id.CacheAge.Seconds())))
	} else {
		h.Del(HeaderCacheAge)
	}
}

// setOrDelete sets h[key]=value when value is non-empty, otherwise deletes
// any existing entry. Keeps the injected header set free of empty strings
// while still overwriting stale values from the inbound request.
func setOrDelete(h http.Header, key, value string) {
	if value == "" {
		h.Del(key)
		return
	}
	h.Set(key, value)
}
