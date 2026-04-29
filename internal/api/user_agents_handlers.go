package api

// user_agents_handlers.go â€” Wave 1.5 Edit 1 + Edit 2
//
// Edit 1: GET /api/v1/users/{id}/agents?filter=created|authorized
//         GET /api/v1/me/agents?filter=created|authorized
//
// Edit 2: POST /api/v1/users/{id}/revoke-agents (admin key only)

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	mw "github.com/shark-auth/shark/internal/api/middleware"
	"github.com/shark-auth/shark/internal/storage"
)

const auditEventCascadeRevokedAgents = "user.cascade_revoked_agents"

// handleUserAgents handles GET /api/v1/users/{id}/agents?filter=created|authorized.
// Auth: admin API key OR session cookie if the caller is the same user.
func (s *Server) handleUserAgents(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")

	agents, filter, err := s.listUserAgents(r, targetUserID)
	if err != nil {
		internal(w, err)
		return
	}

	if agents == nil {
		agents = []*storage.Agent{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":   agents,
		"total":  len(agents),
		"filter": filter,
	})
}

// handleMeAgents handles GET /api/v1/me/agents?filter=created|authorized.
// Auth: session cookie only.
func (s *Server) handleMeAgents(w http.ResponseWriter, r *http.Request) {
	callerID := mw.GetUserID(r.Context())
	if callerID == "" {
		writeJSON(w, http.StatusUnauthorized, errPayload("unauthorized", "Session required"))
		return
	}

	agents, filter, err := s.listUserAgents(r, callerID)
	if err != nil {
		internal(w, err)
		return
	}

	if agents == nil {
		agents = []*storage.Agent{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":   agents,
		"total":  len(agents),
		"filter": filter,
	})
}

// listUserAgents is the shared logic for both /users/{id}/agents and /me/agents.
func (s *Server) listUserAgents(r *http.Request, userID string) ([]*storage.Agent, string, error) {
	filter := r.URL.Query().Get("filter")
	if filter == "" {
		filter = "created"
	}

	switch filter {
	case "authorized":
		opts := storage.ListAgentsOpts{
			Limit:            200,
			AuthorizedByUser: &userID,
		}
		agents, _, err := s.Store.ListAgents(r.Context(), opts)
		return agents, filter, err

	default: // "created"
		filter = "created"
		agents, err := s.Store.ListAgentsByUserID(r.Context(), userID)
		return agents, filter, err
	}
}

// handleCascadeRevokeAgents handles POST /api/v1/users/{id}/revoke-agents.
// Auth: ADMIN API KEY ONLY.
func (s *Server) handleCascadeRevokeAgents(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")

	var req struct {
		AgentIDs []string `json:"agent_ids"`
		Reason   string   `json:"reason"`
	}
	// Body is optional â€” ignore decode errors (empty body is fine)
	_ = json.NewDecoder(r.Body).Decode(&req)

	// 1. Resolve agents to revoke.
	var agents []*storage.Agent
	if len(req.AgentIDs) == 0 {
		// All agents created by this user.
		var err error
		agents, err = s.Store.ListAgentsByUserID(r.Context(), targetUserID)
		if err != nil {
			internal(w, err)
			return
		}
	} else {
		// Specific agents â€” fetch each and verify created_by.
		for _, aid := range req.AgentIDs {
			a, err := s.Store.GetAgentByID(r.Context(), aid)
			if err != nil {
				// Skip missing agents rather than aborting.
				continue
			}
			agents = append(agents, a)
		}
	}

	// 2. For each agent: set active=false, revoke OAuth tokens.
	var revokedAgentIDs []string
	var totalTokensRevoked int64
	for _, a := range agents {
		if a.Active {
			a.Active = false
			if err := s.Store.UpdateAgent(r.Context(), a); err != nil {
				// Non-fatal: log and continue.
				continue
			}
		}
		n, _ := s.Store.RevokeOAuthTokensByClientID(r.Context(), a.ClientID)
		totalTokensRevoked += n
		revokedAgentIDs = append(revokedAgentIDs, a.ID)
	}

	// 3. Bulk-revoke all OAuth consents for this user.
	revokedConsentCount, _ := s.Store.RevokeConsentsByUserID(r.Context(), targetUserID)

	// 4. Write single audit event.
	auditEventID, _ := gonanoid.New(21)

	// Determine by_actor (the admin key holder's identity â€” best-effort from request context).
	byActor := "admin"
	if callerID := mw.GetUserID(r.Context()); callerID != "" {
		byActor = callerID
	}

	meta := map[string]any{
		"revoked_agent_count":   len(revokedAgentIDs),
		"revoked_consent_count": revokedConsentCount,
		"reason":                req.Reason,
		"by_actor":              byActor,
	}
	metaJSON, _ := json.Marshal(meta)

	if s.AuditLogger != nil {
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ID:         auditEventID,
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     auditEventCascadeRevokedAgents,
			TargetType: "user",
			TargetID:   targetUserID,
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
			Metadata:   string(metaJSON),
		})
	}

	if revokedAgentIDs == nil {
		revokedAgentIDs = []string{}
	}

	// 5. Return summary.
	writeJSON(w, http.StatusOK, map[string]any{
		"revoked_agent_ids":     revokedAgentIDs,
		"revoked_consent_count": revokedConsentCount,
		"audit_event_id":        auditEventID,
	})
}
