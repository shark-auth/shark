package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sharkauth/sharkauth/internal/sso"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// ssoStateEntry holds an OIDC state and nonce with an expiry time.
type ssoStateEntry struct {
	connectionID string
	nonce        string
	expiresAt    time.Time
}

// SSOHandlers provides HTTP handlers for SSO endpoints.
type SSOHandlers struct {
	manager    *sso.SSOManager
	server     *Server // for JWT issuance; may be nil before router init
	mu         sync.Mutex
	stateStore map[string]*ssoStateEntry
}

// NewSSOHandlers creates a new SSOHandlers.
func NewSSOHandlers(manager *sso.SSOManager) *SSOHandlers {
	h := &SSOHandlers{
		manager:    manager,
		stateStore: make(map[string]*ssoStateEntry),
	}
	// Clean up expired states every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			h.cleanupStates()
		}
	}()
	return h
}

func (h *SSOHandlers) cleanupStates() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for key, entry := range h.stateStore {
		if now.After(entry.expiresAt) {
			delete(h.stateStore, key)
		}
	}
}

// --- Connection CRUD (admin) ---

// CreateConnection handles POST /api/v1/sso/connections
func (h *SSOHandlers) CreateConnection(w http.ResponseWriter, r *http.Request) {
	var conn storage.SSOConnection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.manager.CreateConnection(r.Context(), &conn); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, conn)
}

// ListConnections handles GET /api/v1/sso/connections
func (h *SSOHandlers) ListConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := h.manager.ListConnections(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, conns)
}

// GetConnection handles GET /api/v1/sso/connections/{id}
func (h *SSOHandlers) GetConnection(w http.ResponseWriter, r *http.Request) {
	id := ssoPathParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing connection id"})
		return
	}

	conn, err := h.manager.GetConnection(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

// UpdateConnection handles PUT /api/v1/sso/connections/{id}
func (h *SSOHandlers) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	id := ssoPathParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing connection id"})
		return
	}

	var conn storage.SSOConnection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	conn.ID = id

	if err := h.manager.UpdateConnection(r.Context(), &conn); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

// DeleteConnection handles DELETE /api/v1/sso/connections/{id}
func (h *SSOHandlers) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id := ssoPathParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing connection id"})
		return
	}

	if err := h.manager.DeleteConnection(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

// --- SAML endpoints (public) ---

// SAMLMetadata handles GET /api/v1/sso/saml/{connection_id}/metadata
func (h *SSOHandlers) SAMLMetadata(w http.ResponseWriter, r *http.Request) {
	connID := ssoPathParam(r, "connection_id")
	if connID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing connection_id"})
		return
	}

	metadata, err := h.manager.GenerateSPMetadata(r.Context(), connID)
	if err != nil {
		slog.Error("SAML metadata generation failed", "connection_id", connID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to generate SAML metadata"})
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write(metadata) //nolint:errcheck
}

// SAMLACS handles POST /api/v1/sso/saml/{connection_id}/acs
func (h *SSOHandlers) SAMLACS(w http.ResponseWriter, r *http.Request) {
	connID := ssoPathParam(r, "connection_id")
	if connID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing connection_id"})
		return
	}

	user, session, err := h.manager.HandleSAMLACS(r.Context(), connID, r)
	if err != nil {
		slog.Error("SAML ACS failed", "connection_id", connID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "SSO authentication failed"})
		return
	}

	resp := map[string]interface{}{
		"user":    user,
		"session": session,
	}
	if h.server != nil && h.server.JWTManager != nil && h.server.Config.Auth.JWT.Enabled {
		if h.server.Config.Auth.JWT.Mode == "access_refresh" {
			access, refresh, jwtErr := h.server.JWTManager.IssueAccessRefreshPair(r.Context(), user, session.ID, session.MFAPassed)
			if jwtErr == nil {
				resp["access_token"] = access
				resp["refresh_token"] = refresh
			}
		} else {
			token, jwtErr := h.server.JWTManager.IssueSessionJWT(r.Context(), user, session.ID, session.MFAPassed)
			if jwtErr == nil {
				resp["token"] = token
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- OIDC endpoints (public) ---

// OIDCAuth handles GET /api/v1/sso/oidc/{connection_id}/auth
func (h *SSOHandlers) OIDCAuth(w http.ResponseWriter, r *http.Request) {
	connID := ssoPathParam(r, "connection_id")
	if connID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing connection_id"})
		return
	}

	redirectURL, state, nonce, err := h.manager.BeginOIDCAuth(r.Context(), connID)
	if err != nil {
		slog.Error("OIDC auth begin failed", "connection_id", connID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to initiate SSO authentication"})
		return
	}

	// Store state -> connectionID + nonce mapping for callback verification
	h.mu.Lock()
	h.stateStore[state] = &ssoStateEntry{
		connectionID: connID,
		nonce:        nonce,
		expiresAt:    time.Now().Add(10 * time.Minute),
	}
	h.mu.Unlock()

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// OIDCCallback handles GET /api/v1/sso/oidc/{connection_id}/callback
func (h *SSOHandlers) OIDCCallback(w http.ResponseWriter, r *http.Request) {
	connID := ssoPathParam(r, "connection_id")
	if connID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing connection_id"})
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		// Check for error response from IdP
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":       errMsg,
				"description": desc,
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing code or state parameter"})
		return
	}

	// Verify state matches
	h.mu.Lock()
	entry, ok := h.stateStore[state]
	if ok {
		delete(h.stateStore, state)
	}
	h.mu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired state"})
		return
	}

	if entry.connectionID != connID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "state does not match connection"})
		return
	}

	user, session, err := h.manager.HandleOIDCCallback(r.Context(), connID, code, state, entry.nonce, r)
	if err != nil {
		slog.Error("OIDC callback failed", "connection_id", connID, "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "SSO authentication failed"})
		return
	}

	resp := map[string]interface{}{
		"user":    user,
		"session": session,
	}
	if h.server != nil && h.server.JWTManager != nil && h.server.Config.Auth.JWT.Enabled {
		if h.server.Config.Auth.JWT.Mode == "access_refresh" {
			access, refresh, jwtErr := h.server.JWTManager.IssueAccessRefreshPair(r.Context(), user, session.ID, session.MFAPassed)
			if jwtErr == nil {
				resp["access_token"] = access
				resp["refresh_token"] = refresh
			}
		} else {
			token, jwtErr := h.server.JWTManager.IssueSessionJWT(r.Context(), user, session.ID, session.MFAPassed)
			if jwtErr == nil {
				resp["token"] = token
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Auto-route (public) ---

// SSOAutoRoute handles GET /api/v1/auth/sso?email=user@corp.com
func (h *SSOHandlers) SSOAutoRoute(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing email parameter"})
		return
	}

	conn, err := h.manager.RouteByEmail(r.Context(), email)
	if err != nil {
		slog.Error("SSO auto-route failed", "email", email, "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "No SSO connection found for this email domain"})
		return
	}

	// Determine redirect URL based on connection type
	var redirectURL string
	switch conn.Type {
	case "oidc":
		redirectURL = fmt.Sprintf("/api/v1/sso/oidc/%s/auth", conn.ID)
	case "saml":
		// For SAML, redirect to the IdP's SSO URL
		if conn.SAMLIdPURL != nil {
			redirectURL = *conn.SAMLIdPURL
		}
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unknown connection type"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connection_id":   conn.ID,
		"connection_type": conn.Type,
		"connection_name": conn.Name,
		"redirect_url":    redirectURL,
	})
}

// ssoPathParam extracts a path parameter using chi.URLParam.
func ssoPathParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}
