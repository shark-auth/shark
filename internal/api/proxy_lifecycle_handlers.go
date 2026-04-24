// Package api — proxy lifecycle admin handlers (PROXYV1_5 §4.9 Lane B).
//
// These routes let admins flip the reverse proxy subsystem on/off at
// runtime without restarting the process. They delegate to the proxy
// Manager (internal/proxy/lifecycle.go) which owns the listener pool
// + state machine. Every handler is 404-safe: when Proxy.Enabled=false
// at boot we never wire a Manager, and the handler returns 404 so the
// dashboard can branch cleanly.

package api

import (
	"net/http"
)

// handleProxyLifecycleStart transitions the proxy Manager from Stopped →
// Running. 409 if already running, 404 when the Manager isn't wired.
func (s *Server) handleProxyLifecycleStart(w http.ResponseWriter, r *http.Request) {
	if s.ProxyManager == nil {
		http.NotFound(w, r)
		return
	}
	if err := s.ProxyManager.Start(r.Context()); err != nil {
		writeJSON(w, http.StatusConflict, errPayload("proxy_start_failed", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": s.ProxyManager.Status()})
}

// handleProxyLifecycleStop transitions the proxy Manager to Stopped.
// Idempotent — stopping a stopped manager returns 200 with current state.
func (s *Server) handleProxyLifecycleStop(w http.ResponseWriter, r *http.Request) {
	if s.ProxyManager == nil {
		http.NotFound(w, r)
		return
	}
	if err := s.ProxyManager.Stop(r.Context()); err != nil {
		writeJSON(w, http.StatusConflict, errPayload("proxy_stop_failed", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": s.ProxyManager.Status()})
}

// handleProxyLifecycleReload rebuilds the listener pool and reloads the
// engine rule set from the DB. Reload = Stop + Start in one critical
// section so no caller can sneak a request into a partly-bound pool.
func (s *Server) handleProxyLifecycleReload(w http.ResponseWriter, r *http.Request) {
	if s.ProxyManager == nil {
		http.NotFound(w, r)
		return
	}
	if err := s.ProxyManager.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusConflict, errPayload("proxy_reload_failed", err.Error()))
		return
	}
	// Also refresh the engine's rule set from the DB so DB-backed
	// overrides layered since the last mutation go live on reload.
	_ = s.refreshProxyEngineFromDB(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"data": s.ProxyManager.Status()})
}

// handleProxyLifecycleStatus returns a point-in-time snapshot of the
// Manager state (stopped/running/reloading + listener count + rules
// loaded + started_at + last_error). Separate route from the legacy
// breaker-stats /proxy/status so existing dashboards don't break.
func (s *Server) handleProxyLifecycleStatus(w http.ResponseWriter, r *http.Request) {
	if s.ProxyManager == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": s.ProxyManager.Status()})
}
