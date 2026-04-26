package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handlePostAgentPolicies persists the agent's may_act delegation policy as a
// JSON blob under agent.Metadata["policies"]. The UI sends a body of the shape
// {"may_act": [{"agent_id":"...", "scope":"..."}]} and the same shape is
// returned (with the agent_id echoed) on success.
func (s *Server) handlePostAgentPolicies(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	agent, err := s.getAgentByIDOrClientID(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Agent not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	var req struct {
		MayAct []map[string]any `json:"may_act"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	if agent.Metadata == nil {
		agent.Metadata = map[string]any{}
	}
	if req.MayAct == nil {
		req.MayAct = []map[string]any{}
	}
	agent.Metadata["policies"] = map[string]any{"may_act": req.MayAct}

	if err := s.Store.UpdateAgent(r.Context(), agent); err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id": agent.ID,
		"may_act":  req.MayAct,
	})
}

// handleGetAgentPolicies returns the persisted policies (or an empty
// may_act list when none are set).
func (s *Server) handleGetAgentPolicies(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	agent, err := s.getAgentByIDOrClientID(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Agent not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	mayAct := []any{}
	if agent.Metadata != nil {
		if p, ok := agent.Metadata["policies"].(map[string]any); ok {
			if m, ok := p["may_act"].([]any); ok {
				mayAct = m
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id": agent.ID,
		"may_act":  mayAct,
	})
}
