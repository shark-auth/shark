// Package api â€” PROXYV1_5 Lane B admin handlers: user tier, branding
// design tokens. Kept in a dedicated file so the v1.5 surface is easy
// to audit against PROXYV1_5.md and so the existing branding/user
// handler files don't grow unbounded.

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/shark-auth/shark/internal/storage"
)

// setUserTierRequest is the PATCH body for /admin/users/{id}/tier.
// Only "free" and "pro" are recognised; anything else rejects with 400
// so typos surface early and the Claims baker never sees a ghost tier.
type setUserTierRequest struct {
	Tier string `json:"tier"`
}

// handleSetUserTier persists the caller-supplied tier into users.metadata
// then re-reads the user for the response. 404 when the user id doesn't
// exist so dashboards can distinguish a missing target from a write
// failure. Emits an audit entry keyed on the actor admin session so the
// same admin-action timeline surfaces tier flips.
func (s *Server) handleSetUserTier(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "user id required"))
		return
	}

	var req setUserTierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	tier := strings.ToLower(strings.TrimSpace(req.Tier))
	if tier != "free" && tier != "pro" {
		writeJSON(w, http.StatusBadRequest,
			errPayload("invalid_tier", `tier must be "free" or "pro"`))
		return
	}

	// Capture old tier before the update so the audit log carries
	// the before/after pair. Best-effort: if the read fails, fall
	// back to an empty old_tier rather than blocking the write.
	oldTier := ""
	if before, err := s.Store.GetUserByID(r.Context(), userID); err == nil && before != nil {
		if before.Metadata != "" {
			var probe map[string]any
			if jerr := json.Unmarshal([]byte(before.Metadata), &probe); jerr == nil {
				if v, ok := probe["tier"].(string); ok {
					oldTier = v
				}
			}
		}
	}

	if err := s.Store.SetUserTier(r.Context(), userID, tier); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "User not found"))
			return
		}
		internal(w, err)
		return
	}

	fresh, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		metaBytes, _ := json.Marshal(map[string]any{
			"tier":     tier,
			"old_tier": oldTier,
			"user_id":  userID,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "user.tier.set",
			TargetType: "user",
			TargetID:   userID,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"user": fresh,
			"tier": tier,
		},
	})
}

// setDesignTokensRequest is the PATCH body for /admin/branding/design-tokens.
// DesignTokens is a free-form JSON object so the dashboard can evolve
// its token tree (colors.*, typography.*, spacing.*, motion.*) without
// requiring a schema migration per field.
type setDesignTokensRequest struct {
	DesignTokens map[string]any `json:"design_tokens"`
}

// handleSetDesignTokens writes the caller-supplied design tokens into
// the global branding row. The storage layer JSON-encodes the map so
// the branding.design_tokens column stays valid TEXT. Returns the
// freshly-read branding row on success; callers never have to re-GET.
func (s *Server) handleSetDesignTokens(w http.ResponseWriter, r *http.Request) {
	var req setDesignTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.DesignTokens == nil {
		req.DesignTokens = map[string]any{}
	}

	fields := map[string]any{"design_tokens": req.DesignTokens}
	if err := s.Store.UpdateBranding(r.Context(), "global", fields); err != nil {
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		tokenKeys := make([]string, 0, len(req.DesignTokens))
		for k := range req.DesignTokens {
			tokenKeys = append(tokenKeys, k)
		}
		actorID := "admin_key"
		metaBytes, _ := json.Marshal(map[string]any{
			"token_keys_changed": tokenKeys,
			"by_admin_key":       actorID,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    actorID,
			Action:     "branding.design_tokens.set",
			TargetType: "branding",
			TargetID:   "global",
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	fresh, err := s.Store.GetBranding(r.Context(), "global")
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"branding":      fresh,
			"design_tokens": req.DesignTokens,
		},
	})
}

