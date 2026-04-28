package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// --- Request/Response types ---

// createAPIKeyRequest is the request body for POST /api/v1/api-keys.
type createAPIKeyRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	RateLimit *int     `json:"rate_limit,omitempty"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

// createAPIKeyResponse is returned ONCE when a key is created.
type createAPIKeyResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Key       string   `json:"key"` // Full key, shown only on creation
	KeyPrefix string   `json:"key_prefix"`
	KeySuffix string   `json:"key_suffix"`
	Scopes    []string `json:"scopes"`
	RateLimit int      `json:"rate_limit"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// apiKeyResponse is returned for list/get (never includes the full key).
type apiKeyResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyDisplay string   `json:"key_display"` // Masked: sk_live_AbCd...xK9f
	KeyPrefix  string   `json:"key_prefix"`
	KeySuffix  string   `json:"key_suffix"`
	Scopes     []string `json:"scopes"`
	RateLimit  int      `json:"rate_limit"`
	ExpiresAt  *string  `json:"expires_at,omitempty"`
	LastUsedAt *string  `json:"last_used_at,omitempty"`
	CreatedAt  string   `json:"created_at"`
	RevokedAt  *string  `json:"revoked_at,omitempty"`
}

// updateAPIKeyRequest is the request body for PATCH /api/v1/api-keys/{id}.
type updateAPIKeyRequest struct {
	Name      *string  `json:"name,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	RateLimit *int     `json:"rate_limit,omitempty"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
}

// --- Handlers ---

// handleCreateAPIKey creates a new M2M API key.
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Name is required",
		})
		return
	}

	if len(req.Scopes) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "At least one scope is required",
		})
		return
	}

	// Validate expires_at if provided
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		if _, err := time.Parse(time.RFC3339, *req.ExpiresAt); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "invalid_request",
				"message": "expires_at must be a valid RFC3339 timestamp",
			})
			return
		}
	}

	// Generate the API key
	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to generate API key",
		})
		return
	}

	// Marshal scopes to JSON
	scopesJSON, err := json.Marshal(req.Scopes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to encode scopes",
		})
		return
	}

	rateLimit := s.Config.APIKeys.DefaultRateLimit
	if req.RateLimit != nil && *req.RateLimit > 0 {
		rateLimit = *req.RateLimit
	}

	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)

	apiKey := &storage.APIKey{
		ID:        "key_" + id,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeySuffix: keySuffix,
		Scopes:    string(scopesJSON),
		RateLimit: rateLimit,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: now,
	}

	if err := s.Store.CreateAPIKey(r.Context(), apiKey); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to create API key",
		})
		return
	}

	if s.AuditLogger != nil {
		meta := map[string]any{
			"key_name": apiKey.Name,
			"scopes":   req.Scopes,
		}
		if req.ExpiresAt != nil && *req.ExpiresAt != "" {
			meta["expires_at"] = *req.ExpiresAt
		}
		metaBytes, _ := json.Marshal(meta)
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "api_key.created",
			TargetType: "api_key",
			TargetID:   apiKey.ID,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       fullKey,
		KeyPrefix: apiKey.KeyPrefix,
		KeySuffix: apiKey.KeySuffix,
		Scopes:    req.Scopes,
		RateLimit: rateLimit,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: now,
	})
}

// handleListAPIKeys returns all API keys wrapped in an api_keys object.
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.Store.ListAPIKeys(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to list API keys",
		})
		return
	}

	list := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		list = append(list, toAPIKeyResponse(k))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"api_keys": list,
	})
}

// handleGetAPIKey returns a single API key's details.
func (s *Server) handleGetAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	key, err := s.Store.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "API key not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to get API key",
		})
		return
	}

	writeJSON(w, http.StatusOK, toAPIKeyResponse(key))
}

// handleUpdateAPIKey updates an API key.
func (s *Server) handleUpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	key, err := s.Store.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "API key not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to get API key",
		})
		return
	}

	if key.RevokedAt != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Cannot update a revoked API key",
		})
		return
	}

	var req updateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.Name != nil {
		key.Name = *req.Name
	}
	if req.Scopes != nil {
		scopesJSON, err := json.Marshal(req.Scopes)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal_error",
				"message": "Failed to encode scopes",
			})
			return
		}
		key.Scopes = string(scopesJSON)
	}
	if req.RateLimit != nil {
		key.RateLimit = *req.RateLimit
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt != "" {
			if _, err := time.Parse(time.RFC3339, *req.ExpiresAt); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error":   "invalid_request",
					"message": "expires_at must be a valid RFC3339 timestamp",
				})
				return
			}
		}
		key.ExpiresAt = req.ExpiresAt
	}

	if err := s.Store.UpdateAPIKey(r.Context(), key); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to update API key",
		})
		return
	}

	writeJSON(w, http.StatusOK, toAPIKeyResponse(key))
}

// handleRevokeAPIKey soft-deletes an API key.
func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	key, err := s.Store.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "API key not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to get API key",
		})
		return
	}

	if key.RevokedAt != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "API key is already revoked",
		})
		return
	}

	// Prevent revoking the last admin key (scope "*")
	var scopes []string
	_ = json.Unmarshal([]byte(key.Scopes), &scopes)
	if auth.CheckScope(scopes, "*") {
		count, err := s.Store.CountActiveAPIKeysByScope(r.Context(), "*")
		if err == nil && count <= 1 {
			writeJSON(w, http.StatusConflict, map[string]string{
				"error":   "last_admin_key",
				"message": "Cannot revoke the last admin API key.",
			})
			return
		}
	}

	now := time.Now().UTC()
	if err := s.Store.RevokeAPIKey(r.Context(), id, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to revoke API key",
		})
		return
	}

	if s.AuditLogger != nil {
		actorID := "admin_key"
		metaBytes, _ := json.Marshal(map[string]any{
			"key_name":       key.Name,
			"revoked_by_kid": actorID,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    actorID,
			Action:     "api_key.revoked",
			TargetType: "api_key",
			TargetID:   id,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	revokedAt := now.Format(time.RFC3339)
	key.RevokedAt = &revokedAt

	writeJSON(w, http.StatusOK, toAPIKeyResponse(key))
}

// handleHardDeleteAPIKey permanently deletes an API key.
func (s *Server) handleHardDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	key, err := s.Store.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "API key not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to get API key",
		})
		return
	}

	// Prevent deleting the last admin key (scope "*")
	var scopes []string
	_ = json.Unmarshal([]byte(key.Scopes), &scopes)
	if auth.CheckScope(scopes, "*") {
		count, err := s.Store.CountActiveAPIKeysByScope(r.Context(), "*")
		if err == nil && count <= 1 {
			// Even if revoked, if it's somehow the last one returned by CountActive (unlikely if revoked, but safe to check)
			// Wait, CountActiveAPIKeysByScope only counts NON-revoked keys. So if THIS key is revoked, count is already not including it.
			// But just in case this key is NOT revoked, we shouldn't let them hard delete it if it's the last active one.
			if key.RevokedAt == nil && count <= 1 {
				writeJSON(w, http.StatusConflict, map[string]string{
					"error":   "last_admin_key",
					"message": "Cannot delete the last active admin API key.",
				})
				return
			}
		}
	}

	if err := s.Store.DeleteAPIKey(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to delete API key",
		})
		return
	}

	if s.AuditLogger != nil {
		actorID := "admin_key"
		metaBytes, _ := json.Marshal(map[string]any{
			"key_name":       key.Name,
			"deleted_by_kid": actorID,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    actorID,
			Action:     "api_key.deleted",
			TargetType: "api_key",
			TargetID:   id,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleRotateAPIKey rotates an API key.
func (s *Server) handleRotateAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	oldKey, err := s.Store.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "API key not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to get API key",
		})
		return
	}

	if oldKey.RevokedAt != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Cannot rotate a revoked API key",
		})
		return
	}

	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to generate new API key",
		})
		return
	}

	newID, _ := gonanoid.New()
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	newKey := &storage.APIKey{
		ID:        "key_" + newID,
		Name:      oldKey.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeySuffix: keySuffix,
		Scopes:    oldKey.Scopes,
		RateLimit: oldKey.RateLimit,
		ExpiresAt: oldKey.ExpiresAt,
		CreatedAt: nowStr,
	}

	if err := s.Store.CreateAPIKey(r.Context(), newKey); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to create rotated API key",
		})
		return
	}

	if err := s.Store.RevokeAPIKey(r.Context(), id, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to revoke old API key",
		})
		return
	}

	var scopes []string
	_ = json.Unmarshal([]byte(newKey.Scopes), &scopes)

	if s.AuditLogger != nil {
		metaBytes, _ := json.Marshal(map[string]any{
			"old_key_id": oldKey.ID,
			"new_key_id": newKey.ID,
			"scopes":     scopes,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "api_key.rotated",
			TargetType: "api_key",
			TargetID:   newKey.ID,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{
		ID:        newKey.ID,
		Name:      newKey.Name,
		Key:       fullKey,
		KeyPrefix: newKey.KeyPrefix,
		KeySuffix: newKey.KeySuffix,
		Scopes:    scopes,
		RateLimit: newKey.RateLimit,
		ExpiresAt: newKey.ExpiresAt,
		CreatedAt: nowStr,
	})
}

// --- Helpers ---

func toAPIKeyResponse(k *storage.APIKey) apiKeyResponse {
	var scopes []string
	_ = json.Unmarshal([]byte(k.Scopes), &scopes)

	display := "sk_live_" + k.KeyPrefix + "..." + k.KeySuffix

	return apiKeyResponse{
		ID:         k.ID,
		Name:       k.Name,
		KeyDisplay: display,
		KeyPrefix:  k.KeyPrefix,
		KeySuffix:  k.KeySuffix,
		Scopes:     scopes,
		RateLimit:  k.RateLimit,
		ExpiresAt:  k.ExpiresAt,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
		RevokedAt:  k.RevokedAt,
	}
}
