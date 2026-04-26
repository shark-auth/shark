package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/storage"
)

type consentResponse struct {
	ID                   string     `json:"id"`
	ClientID             string     `json:"client_id"`
	AgentName            string     `json:"agent_name,omitempty"`
	Scope                string     `json:"scope"`
	AuthorizationDetails string     `json:"authorization_details,omitempty"`
	GrantedAt            time.Time  `json:"granted_at"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
}

// handleListConsents handles GET /api/v1/auth/consents.
// Returns the authenticated user's active (non-revoked) consent grants.
func (s *Server) handleListConsents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := mw.GetUserID(ctx)

	consents, err := s.Store.ListConsentsByUserID(ctx, userID)
	if err != nil {
		internal(w, err)
		return
	}

	resp := make([]consentResponse, 0, len(consents))
	for _, c := range consents {
		cr := consentResponse{
			ID:                   c.ID,
			ClientID:             c.ClientID,
			Scope:                c.Scope,
			AuthorizationDetails: c.AuthorizationDetails,
			GrantedAt:            c.GrantedAt,
			ExpiresAt:            c.ExpiresAt,
		}
		// Best-effort: enrich with agent name; ignore error if agent not found.
		if agent, err := s.Store.GetAgentByClientID(ctx, c.ClientID); err == nil {
			cr.AgentName = agent.Name
		}
		resp = append(resp, cr)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": resp})
}

// handleRevokeConsent handles DELETE /api/v1/auth/consents/{id}.
// Verifies ownership (IDOR protection), revokes the consent, and revokes all
// OAuth tokens issued to that client for this user.
func (s *Server) handleRevokeConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := mw.GetUserID(ctx)
	consentID := chi.URLParam(r, "id")

	// Fetch all consents for this user and find the matching one.
	// This simultaneously verifies the consent exists AND belongs to the caller.
	consents, err := s.Store.ListConsentsByUserID(ctx, userID)
	if err != nil {
		internal(w, err)
		return
	}

	var target *storage.OAuthConsent
	for _, c := range consents {
		if c.ID == consentID {
			target = c
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Consent not found",
		})
		return
	}

	// Revoke the consent record.
	if err := s.Store.RevokeOAuthConsent(ctx, consentID); err != nil {
		internal(w, err)
		return
	}

	// Revoke all OAuth tokens issued to this client (best-effort; non-fatal).
	if _, err := s.Store.RevokeOAuthTokensByClientID(ctx, target.ClientID); err != nil {
		_ = err // log silently; consent is already revoked
	}

	// Audit log the revocation.
	if s.AuditLogger != nil {
		meta, _ := json.Marshal(map[string]string{
			"consent_id": consentID,
			"client_id":  target.ClientID,
			"user_id":    userID,
		})
		_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
			ActorID:    userID,
			ActorType:  "user",
			Action:     "consent.revoked",
			TargetType: "consent",
			TargetID:   consentID,
			IP:         ipOf(r),
			UserAgent:  uaOf(r),
			Metadata:   string(meta),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Consent revoked"})
}

// adminConsentResponse is the cross-user consent view used by the admin
// dashboard. Adds user_id on top of the user-facing fields so operators can
// triage consents across the whole tenant.
type adminConsentResponse struct {
	consentResponse
	UserID string `json:"user_id"`
}

// handleAdminListConsents handles GET /api/v1/admin/oauth/consents.
// Returns every active OAuth consent across all users (admin scope). The
// per-user /auth/consents endpoint above is session-scoped to the caller.
// Accepts optional ?user_id=<id> to filter to a single user's consents.
func (s *Server) handleAdminListConsents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var consents []*storage.OAuthConsent
	var err error
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		consents, err = s.Store.ListConsentsByUserID(ctx, uid)
	} else {
		consents, err = s.Store.ListAllConsents(ctx)
	}
	if err != nil {
		internal(w, err)
		return
	}
	resp := make([]adminConsentResponse, 0, len(consents))
	agentCache := make(map[string]string)
	for _, c := range consents {
		row := adminConsentResponse{
			consentResponse: consentResponse{
				ID:                   c.ID,
				ClientID:             c.ClientID,
				Scope:                c.Scope,
				AuthorizationDetails: c.AuthorizationDetails,
				GrantedAt:            c.GrantedAt,
				ExpiresAt:            c.ExpiresAt,
			},
			UserID: c.UserID,
		}
		if name, ok := agentCache[c.ClientID]; ok {
			row.AgentName = name
		} else if agent, err := s.Store.GetAgentByClientID(ctx, c.ClientID); err == nil {
			row.AgentName = agent.Name
			agentCache[c.ClientID] = agent.Name
		}
		resp = append(resp, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": resp, "total": len(resp)})
}

// handleAdminRevokeConsent handles DELETE /api/v1/admin/oauth/consents/{id}.
// Cross-user revoke. Skips ownership check (admin override) and audits as
// admin actor. Same token-cascade revocation as the user-facing handler.
func (s *Server) handleAdminRevokeConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := chi.URLParam(r, "id")

	// We need the row to capture client_id + user_id for audit + cascade.
	all, err := s.Store.ListAllConsents(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	var target *storage.OAuthConsent
	for _, c := range all {
		if c.ID == consentID {
			target = c
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Consent not found"))
		return
	}

	if err := s.Store.RevokeOAuthConsent(ctx, consentID); err != nil {
		internal(w, err)
		return
	}
	if _, err := s.Store.RevokeOAuthTokensByClientID(ctx, target.ClientID); err != nil {
		_ = err
	}
	if s.AuditLogger != nil {
		meta, _ := json.Marshal(map[string]string{
			"consent_id": consentID,
			"client_id":  target.ClientID,
			"user_id":    target.UserID,
		})
		_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
			ActorType:  "admin",
			Action:     "consent.revoked",
			TargetType: "consent",
			TargetID:   consentID,
			IP:         ipOf(r),
			UserAgent:  uaOf(r),
			Metadata:   string(meta),
			Status:     "success",
		})
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Consent revoked"})
}

// handleAdminGrantConsent handles POST /api/v1/admin/consents.
// Admin-key auth. Inserts (or returns existing) oauth_consents row.
// Body: {"user_id":"...","client_id":"...","scopes":["..."]}
func (s *Server) handleAdminGrantConsent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID   string   `json:"user_id"`
		ClientID string   `json:"client_id"`
		Scopes   []string `json:"scopes"`
		Scope    string   `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if body.UserID == "" || body.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "user_id and client_id are required"))
		return
	}
	scopeStr := body.Scope
	if scopeStr == "" {
		scopeStr = strings.Join(body.Scopes, " ")
	}

	ctx := r.Context()
	if existing, err := s.Store.GetActiveConsent(ctx, body.UserID, body.ClientID); err == nil && existing != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}

	consent := &storage.OAuthConsent{
		ID:        "consent_" + body.ClientID + "_" + body.UserID,
		UserID:    body.UserID,
		ClientID:  body.ClientID,
		Scope:     scopeStr,
		GrantedAt: time.Now().UTC(),
	}
	if err := s.Store.CreateOAuthConsent(ctx, consent); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, consent)
}
