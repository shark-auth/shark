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
	Scopes    []string `json:"scopes"`
	RateLimit int      `json:"rate_limit"`
	ExpiresAt *string  `json:"expires_at,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// apiKeyResponse is returned for list/get (never includes the full key).
type apiKeyResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	KeyPrefix  string   `json:"key_prefix"`
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

// handleCreateAPIKey creates a new M2M API key. The full key is returned
// once in the response and never stored or shown again.
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
	fullKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
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

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       fullKey,
		KeyPrefix: apiKey.KeyPrefix,
		Scopes:    req.Scopes,
		RateLimit: rateLimit,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: now,
	})
}

// handleListAPIKeys returns all API keys (prefix + metadata, never the full key).
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.Store.ListAPIKeys(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to list API keys",
		})
		return
	}

	resp := make([]apiKeyResponse, 0, len(keys))
	for _, k := range keys {
		resp = append(resp, toAPIKeyResponse(k))
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleGetAPIKey returns a single API key's details (no full key).
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

// handleUpdateAPIKey updates name, scopes, rate_limit, or expires_at on an API key.
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

	// Cannot update a revoked key
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

// handleRevokeAPIKey soft-deletes an API key by setting revoked_at.
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

	now := time.Now().UTC()
	if err := s.Store.RevokeAPIKey(r.Context(), id, now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to revoke API key",
		})
		return
	}

	revokedAt := now.Format(time.RFC3339)
	key.RevokedAt = &revokedAt

	writeJSON(w, http.StatusOK, toAPIKeyResponse(key))
}

// handleRotateAPIKey atomically creates a new key and revokes the old one.
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

	// Generate new key
	fullKey, keyHash, keyPrefix, err := auth.GenerateAPIKey()
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
		Scopes:    oldKey.Scopes,
		RateLimit: oldKey.RateLimit,
		ExpiresAt: oldKey.ExpiresAt,
		CreatedAt: nowStr,
	}

	// Create new key first, then revoke old one
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

	// Parse scopes for response
	var scopes []string
	_ = json.Unmarshal([]byte(newKey.Scopes), &scopes)

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{
		ID:        newKey.ID,
		Name:      newKey.Name,
		Key:       fullKey,
		KeyPrefix: newKey.KeyPrefix,
		Scopes:    scopes,
		RateLimit: newKey.RateLimit,
		ExpiresAt: newKey.ExpiresAt,
		CreatedAt: nowStr,
	})
}

// --- Helpers ---

// toAPIKeyResponse converts a storage.APIKey to the public response type.
// Never includes the key hash or the full key.
func toAPIKeyResponse(k *storage.APIKey) apiKeyResponse {
	var scopes []string
	_ = json.Unmarshal([]byte(k.Scopes), &scopes)

	return apiKeyResponse{
		ID:         k.ID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		Scopes:     scopes,
		RateLimit:  k.RateLimit,
		ExpiresAt:  k.ExpiresAt,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
		RevokedAt:  k.RevokedAt,
	}
}

