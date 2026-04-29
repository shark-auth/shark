package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/shark-auth/shark/internal/storage"
)

// createMayActGrantRequest is the POST body for /api/v1/admin/may-act.
type createMayActGrantRequest struct {
	FromID    string   `json:"from_id"`
	ToID      string   `json:"to_id"`
	MaxHops   int      `json:"max_hops"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at,omitempty"`
}

// handleListMayActGrants â€” GET /api/v1/admin/may-act
//
// Query params: from_id, to_id, include_revoked.
// Response: { "grants": [ ...MayActGrant ] }.
func (s *Server) handleListMayActGrants(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := storage.ListMayActGrantsQuery{
		FromID:         strings.TrimSpace(q.Get("from_id")),
		ToID:           strings.TrimSpace(q.Get("to_id")),
		IncludeRevoked: q.Get("include_revoked") == "true",
	}
	grants, err := s.Store.ListMayActGrants(r.Context(), opts)
	if err != nil {
		internal(w, err)
		return
	}
	if grants == nil {
		grants = []*storage.MayActGrant{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"grants": grants})
}

// handleCreateMayActGrant â€” POST /api/v1/admin/may-act
// Operator-issued grant insertion. Used by tests + future admin UI.
func (s *Server) handleCreateMayActGrant(w http.ResponseWriter, r *http.Request) {
	var req createMayActGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.FromID == "" || req.ToID == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "'from_id' and 'to_id' are required"))
		return
	}
	if req.MaxHops <= 0 {
		req.MaxHops = 1
	}
	if req.ExpiresAt != "" {
		if _, err := time.Parse(time.RFC3339, req.ExpiresAt); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_parameter", "expires_at must be RFC3339"))
			return
		}
	}
	id, _ := gonanoid.New(21)
	grant := &storage.MayActGrant{
		ID:      "mag_" + id,
		FromID:  req.FromID,
		ToID:    req.ToID,
		MaxHops: req.MaxHops,
		Scopes:  req.Scopes,
	}
	if req.ExpiresAt != "" {
		ev := req.ExpiresAt
		grant.ExpiresAt = &ev
	}
	if err := s.Store.CreateMayActGrant(r.Context(), grant); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, grant)
}

// handleRevokeMayActGrant â€” DELETE /api/v1/admin/may-act/{id}
// Sets revoked_at; idempotent (already-revoked grants return the current row).
func (s *Server) handleRevokeMayActGrant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Missing grant id"))
		return
	}
	existing, err := s.Store.GetMayActGrantByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Grant not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if existing.RevokedAt == nil {
		if err := s.Store.RevokeMayActGrant(r.Context(), id, time.Now().UTC()); err != nil {
			internal(w, err)
			return
		}
		// Re-read so the response reflects the revoked_at value.
		existing, err = s.Store.GetMayActGrantByID(r.Context(), id)
		if err != nil {
			internal(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, existing)
}
