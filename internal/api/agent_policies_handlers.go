package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handlePostAgentPolicies persists the agent's delegation policy as a JSON blob
// under agent.Metadata["policies"]. Accepts either:
//
//	{"may_act":  [{"agent_id":"...",       "scope":"..."}]}   // dashboard shape
//	{"policies": [{"delegate_to_id":"...", "scope":"..."}]}   // smoke-test shape
//
// Either field with an empty array clears all delegations.
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

	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	var mayAct []any
	if v, ok := raw["may_act"]; ok {
		arr, isArr := v.([]any)
		if !isArr {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "'may_act' must be an array"))
			return
		}
		mayAct = arr
	} else if v, ok := raw["policies"]; ok {
		arr, isArr := v.([]any)
		if !isArr {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "'policies' must be an array"))
			return
		}
		// Normalise {delegate_to_id,scope} → {agent_id,scope}
		normalised := make([]any, 0, len(arr))
		for _, item := range arr {
			obj, ok := item.(map[string]any)
			if !ok {
				normalised = append(normalised, item)
				continue
			}
			if _, has := obj["agent_id"]; !has {
				if did, ok := obj["delegate_to_id"]; ok {
					obj["agent_id"] = did
				}
			}
			normalised = append(normalised, obj)
		}
		mayAct = normalised
	} else {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Body must include 'may_act' or 'policies' (array; pass [] to clear)"))
		return
	}

	if agent.Metadata == nil {
		agent.Metadata = map[string]any{}
	}
	agent.Metadata["policies"] = map[string]any{"may_act": mayAct}

	if err := s.Store.UpdateAgent(r.Context(), agent); err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agent_id": agent.ID,
		"may_act":  mayAct,
		"policies": mayAct,
	})
}

// handleGetAgentPolicies returns the persisted policies (or an empty list when
// none are set). Echoes both `may_act` and `policies` for compatibility.
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
		"policies": mayAct,
		"data":     mayAct,
	})
}
