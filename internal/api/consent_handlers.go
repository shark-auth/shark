package api

import (
	"encoding/json"
	"net/http"
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
