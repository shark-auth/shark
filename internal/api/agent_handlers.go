package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// agentCreateResponse wraps an Agent and includes the one-time client_secret.
type agentCreateResponse struct {
	storage.Agent
	ClientSecret string `json:"client_secret"`
}

// generateAgentSecret produces a hex-encoded 32-byte random secret and its SHA-256 hash.
func generateAgentSecret() (secret, secretHash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	secret = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(secret))
	secretHash = hex.EncodeToString(h[:])
	return
}

// auditAgent logs an agent-related audit event.
func (s *Server) auditAgent(r *http.Request, action, targetID string) {
	if s.AuditLogger == nil {
		return
	}
	_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
		ActorType:  "admin",
		Action:     action,
		TargetType: "agent",
		TargetID:   targetID,
		IP:         r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		Status:     "success",
	})
}

// auditAgentWithMeta logs an agent audit event with additional metadata fields.
func (s *Server) auditAgentWithMeta(r *http.Request, action, targetID string, meta map[string]any) {
	if s.AuditLogger == nil {
		return
	}
	metaJSON, _ := json.Marshal(meta)
	_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
		ActorType:  "admin",
		Action:     action,
		TargetType: "agent",
		TargetID:   targetID,
		IP:         r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		Status:     "success",
		Metadata:   string(metaJSON),
	})
}

// emitAgentEvent emits a webhook event if the dispatcher is wired.
func (s *Server) emitAgentEvent(r *http.Request, event string, payload any) {
	if s.WebhookDispatcher == nil {
		return
	}
	_ = s.WebhookDispatcher.Emit(r.Context(), event, payload)
}

// POST /api/v1/agents
func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string         `json:"name"`
		Description   string         `json:"description"`
		ClientType    string         `json:"client_type"`
		AuthMethod    string         `json:"auth_method"`
		RedirectURIs  []string       `json:"redirect_uris"`
		AllowedCallbackURLs []string `json:"allowed_callback_urls"` // Alias for RedirectURIs
		GrantTypes    []string       `json:"grant_types"`
		ResponseTypes []string       `json:"response_types"`
		Scopes        []string       `json:"scopes"`
		TokenLifetime int            `json:"token_lifetime"`
		Metadata      map[string]any `json:"metadata"`
		LogoURI       string         `json:"logo_uri"`
		HomepageURI   string         `json:"homepage_uri"`
		CreatedBy     string         `json:"created_by"` // W1.5: explicit creator-user binding for cascade-revoke
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "name is required"))
		return
	}

	// Defaults
	if req.ClientType == "" {
		req.ClientType = "confidential"
	}
	if req.AuthMethod == "" {
		req.AuthMethod = "client_secret_basic"
	}
	if req.TokenLifetime == 0 {
		req.TokenLifetime = 3600
	}

	nid, err := gonanoid.New(21)
	if err != nil {
		internal(w, err)
		return
	}
	agentNid, err := gonanoid.New()
	if err != nil {
		internal(w, err)
		return
	}

	secret, secretHash, err := generateAgentSecret()
	if err != nil {
		internal(w, err)
		return
	}

	now := time.Now().UTC()

	redirectURIs := req.RedirectURIs
	if redirectURIs == nil {
		redirectURIs = []string{}
	}
	if len(req.AllowedCallbackURLs) > 0 {
		redirectURIs = append(redirectURIs, req.AllowedCallbackURLs...)
	}
	grantTypes := req.GrantTypes
	if grantTypes == nil {
		grantTypes = []string{}
	}
	responseTypes := req.ResponseTypes
	if responseTypes == nil {
		responseTypes = []string{}
	}
	scopes := req.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	agent := &storage.Agent{
		ID:               "agent_" + agentNid,
		Name:             req.Name,
		Description:      req.Description,
		ClientID:         "shark_agent_" + nid,
		ClientSecretHash: secretHash,
		ClientType:       req.ClientType,
		AuthMethod:       req.AuthMethod,
		RedirectURIs:     redirectURIs,
		GrantTypes:       grantTypes,
		ResponseTypes:    responseTypes,
		Scopes:           scopes,
		TokenLifetime:    req.TokenLifetime,
		Metadata:         metadata,
		LogoURI:          req.LogoURI,
		HomepageURI:      req.HomepageURI,
		CreatedBy:        req.CreatedBy,
		Active:           true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.Store.CreateAgent(r.Context(), agent); err != nil {
		internal(w, err)
		return
	}

	s.auditAgent(r, "agent.created", agent.ID)
	s.emitAgentEvent(r, "agent.created", agent)

	writeJSON(w, http.StatusCreated, agentCreateResponse{
		Agent:        *agent,
		ClientSecret: secret,
	})
}

// GET /api/v1/agents
func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 50
	offset := 0
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	opts := storage.ListAgentsOpts{
		Limit:  limit,
		Offset: offset,
		Search: q.Get("search"),
	}

	if v := q.Get("active"); v != "" {
		b := v == "true" || v == "1"
		opts.Active = &b
	}
	if v := q.Get("created_by_user_id"); v != "" {
		opts.CreatedByUserID = &v
	}

	agents, total, err := s.Store.ListAgents(r.Context(), opts)
	if err != nil {
		internal(w, err)
		return
	}

	if agents == nil {
		agents = []*storage.Agent{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  agents,
		"total": total,
	})
}

// GET /api/v1/agents/{id}
func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, agent)
}

// PATCH /api/v1/agents/{id}
func (s *Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
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
		Name          *string         `json:"name"`
		Description   *string         `json:"description"`
		RedirectURIs  *[]string       `json:"redirect_uris"`
		AllowedCallbackURLs *[]string `json:"allowed_callback_urls"` // Alias for RedirectURIs
		GrantTypes    *[]string       `json:"grant_types"`
		Scopes        *[]string       `json:"scopes"`
		TokenLifetime *int            `json:"token_lifetime"`
		Metadata      *map[string]any `json:"metadata"`
		LogoURI       *string         `json:"logo_uri"`
		HomepageURI   *string         `json:"homepage_uri"`
		Active        *bool           `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	if req.Name != nil {
		agent.Name = *req.Name
	}
	if req.Description != nil {
		agent.Description = *req.Description
	}
	if req.RedirectURIs != nil {
		agent.RedirectURIs = *req.RedirectURIs
	}
	if req.AllowedCallbackURLs != nil {
		agent.RedirectURIs = append(agent.RedirectURIs, *req.AllowedCallbackURLs...)
	}
	if req.GrantTypes != nil {
		agent.GrantTypes = *req.GrantTypes
	}
	if req.Scopes != nil {
		agent.Scopes = *req.Scopes
	}
	if req.TokenLifetime != nil {
		agent.TokenLifetime = *req.TokenLifetime
	}
	if req.Metadata != nil {
		agent.Metadata = *req.Metadata
	}
	if req.LogoURI != nil {
		agent.LogoURI = *req.LogoURI
	}
	if req.HomepageURI != nil {
		agent.HomepageURI = *req.HomepageURI
	}
	deactivating := req.Active != nil && !*req.Active && agent.Active
	if req.Active != nil {
		agent.Active = *req.Active
	}

	if err := s.Store.UpdateAgent(r.Context(), agent); err != nil {
		internal(w, err)
		return
	}

	if deactivating {
		// Revoke all existing tokens so the UI promise is kept:
		// "Deactivating will prevent new tokens and revoke all active tokens."
		revokedCount, revokeErr := s.Store.RevokeOAuthTokensByClientID(r.Context(), agent.ClientID)
		_ = revokeErr // non-fatal; agent already deactivated
		s.auditAgentWithMeta(r, "agent.deactivated_with_revocation", agent.ID, map[string]any{
			"revoked_token_count": revokedCount,
		})
	} else {
		s.auditAgent(r, "agent.updated", agent.ID)
	}
	s.emitAgentEvent(r, "agent.updated", agent)

	// Re-fetch to pick up updated_at set by the DB.
	updated, err := s.Store.GetAgentByID(r.Context(), agent.ID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// DELETE /api/v1/agents/{id}
func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
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

	if err := s.Store.DeactivateAgent(r.Context(), agent.ID); err != nil {
		internal(w, err)
		return
	}

	if _, err := s.Store.RevokeOAuthTokensByClientID(r.Context(), agent.ClientID); err != nil {
		// Non-fatal: log but continue; agent is already deactivated.
		_ = err
	}

	s.auditAgent(r, "agent.deactivated", agent.ID)
	s.emitAgentEvent(r, "agent.deactivated", map[string]string{"id": agent.ID, "client_id": agent.ClientID})

	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/agents/{id}/tokens
func (s *Server) handleListAgentTokens(w http.ResponseWriter, r *http.Request) {
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

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	tokens, err := s.Store.ListOAuthTokensByAgentID(r.Context(), agent.ID, limit)
	if err != nil {
		internal(w, err)
		return
	}

	if tokens == nil {
		tokens = []*storage.OAuthToken{}
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": tokens})
}

// POST /api/v1/agents/{id}/tokens/revoke-all
func (s *Server) handleRevokeAgentTokens(w http.ResponseWriter, r *http.Request) {
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

	count, err := s.Store.RevokeOAuthTokensByClientID(r.Context(), agent.ClientID)
	if err != nil {
		internal(w, err)
		return
	}

	s.auditAgent(r, "agent.tokens_revoked", agent.ID)

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Tokens revoked",
		"count":   count,
	})
}

// POST /api/v1/agents/{id}/rotate-secret
// Generates a fresh client secret, stores its hash, and returns the plaintext
// exactly once. Requires admin key. Emits audit log entry.
func (s *Server) handleAgentRotateSecret(w http.ResponseWriter, r *http.Request) {
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

	secret, secretHash, err := generateAgentSecret()
	if err != nil {
		internal(w, err)
		return
	}

	if err := s.Store.UpdateAgentSecret(r.Context(), agent.ID, secretHash); err != nil {
		internal(w, err)
		return
	}

	s.auditAgent(r, "agent.secret.rotated", agent.ID)

	writeJSON(w, http.StatusOK, map[string]string{
		"client_id":     agent.ClientID,
		"client_secret": secret,
		"message":       "Secret rotated. Copy it now — it will not be shown again.",
	})
}

// GET /api/v1/agents/{id}/audit
func (s *Server) handleAgentAuditLogs(w http.ResponseWriter, r *http.Request) {
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

	q := r.URL.Query()

	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}

	cursor := q.Get("cursor")

	// Query logs where agent is target
	targetLogs, err := s.AuditLogger.Query(r.Context(), storage.AuditLogQuery{
		TargetID: agent.ID,
		Limit:    limit,
		Cursor:   cursor,
	})
	if err != nil {
		internal(w, err)
		return
	}

	// Query logs where agent is actor
	actorLogs, err := s.AuditLogger.Query(r.Context(), storage.AuditLogQuery{
		ActorID: agent.ID,
		Limit:   limit,
		Cursor:  cursor,
	})
	if err != nil {
		internal(w, err)
		return
	}

	// Merge and deduplicate
	seen := make(map[string]bool)
	var merged []*storage.AuditLog
	for _, l := range targetLogs {
		if !seen[l.ID] {
			seen[l.ID] = true
			merged = append(merged, l)
		}
	}
	for _, l := range actorLogs {
		if !seen[l.ID] {
			seen[l.ID] = true
			merged = append(merged, l)
		}
	}

	sortAuditLogsByCreatedAtDesc(merged)

	if len(merged) > limit {
		merged = merged[:limit]
	}

	if merged == nil {
		merged = []*storage.AuditLog{}
	}

	hasMore := len(targetLogs) >= limit || len(actorLogs) >= limit
	resp := auditListResponse{
		Data:    merged,
		HasMore: hasMore,
	}
	if hasMore && len(merged) > 0 {
		resp.NextCursor = merged[len(merged)-1].ID
	}

	writeJSON(w, http.StatusOK, resp)
}

// getAgentByIDOrClientID tries GetAgentByID then GetAgentByClientID.
func (s *Server) getAgentByIDOrClientID(r *http.Request, idParam string) (*storage.Agent, error) {
	agent, err := s.Store.GetAgentByID(r.Context(), idParam)
	if err == nil {
		return agent, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return s.Store.GetAgentByClientID(r.Context(), idParam)
}

// computeJWKThumbprint computes an RFC 7638 SHA-256 JWK thumbprint and returns
// it as a base64url-encoded string.  Only EC P-256 keys are supported (the
// only curve SharkAuth agents use).  The canonical member set is {crv, kty, x, y}
// sorted lexicographically per RFC 7638 §3.3.
func computeJWKThumbprint(jwk map[string]any) (string, error) {
	kty, _ := jwk["kty"].(string)
	switch kty {
	case "EC":
		crv, _ := jwk["crv"].(string)
		x, _ := jwk["x"].(string)
		y, _ := jwk["y"].(string)
		if crv == "" || x == "" || y == "" {
			return "", fmt.Errorf("EC JWK missing crv/x/y")
		}
		// Canonical JSON: members sorted, no spaces.
		canonical := fmt.Sprintf(`{"crv":%q,"kty":%q,"x":%q,"y":%q}`, crv, kty, x, y)
		sum := sha256.Sum256([]byte(canonical))
		return base64.RawURLEncoding.EncodeToString(sum[:]), nil
	case "RSA":
		e, _ := jwk["e"].(string)
		n, _ := jwk["n"].(string)
		if e == "" || n == "" {
			return "", fmt.Errorf("RSA JWK missing e/n")
		}
		canonical := fmt.Sprintf(`{"e":%q,"kty":%q,"n":%q}`, e, kty, n)
		sum := sha256.Sum256([]byte(canonical))
		return base64.RawURLEncoding.EncodeToString(sum[:]), nil
	case "OKP":
		crv, _ := jwk["crv"].(string)
		x, _ := jwk["x"].(string)
		if crv == "" || x == "" {
			return "", fmt.Errorf("OKP JWK missing crv/x")
		}
		canonical := fmt.Sprintf(`{"crv":%q,"kty":%q,"x":%q}`, crv, kty, x)
		sum := sha256.Sum256([]byte(canonical))
		return base64.RawURLEncoding.EncodeToString(sum[:]), nil
	default:
		// Generic fallback: marshal all public members sorted.
		public := make(map[string]string)
		for k, v := range jwk {
			// Skip private key fields (d, p, q, dp, dq, qi, k)
			switch k {
			case "d", "p", "q", "dp", "dq", "qi", "k":
				continue
			}
			if s, ok := v.(string); ok {
				public[k] = s
			}
		}
		keys := make([]string, 0, len(public))
		for k := range public {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b, _ := json.Marshal(public)
		sum := sha256.Sum256(b)
		_ = keys // used implicitly via json.Marshal which sorts map keys in Go 1.12+
		return base64.RawURLEncoding.EncodeToString(sum[:]), nil
	}
}

// POST /api/v1/agents/{id}/rotate-dpop-key
// Body: {"new_public_jwk": {...}, "reason": "..."}
// Response: {old_jkt, new_jkt, revoked_token_count, audit_event_id}
func (s *Server) handleRotateAgentDPoPKey(w http.ResponseWriter, r *http.Request) {
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
		NewPublicJWK map[string]any `json:"new_public_jwk"`
		Reason       string         `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if len(req.NewPublicJWK) == 0 {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "new_public_jwk is required"))
		return
	}

	// Compute old_jkt from agent Metadata["dpop_public_jwk"], if present.
	oldJKT := ""
	if agent.Metadata != nil {
		if oldJWKRaw, ok := agent.Metadata["dpop_public_jwk"]; ok {
			var oldJWK map[string]any
			switch v := oldJWKRaw.(type) {
			case map[string]any:
				oldJWK = v
			case string:
				_ = json.Unmarshal([]byte(v), &oldJWK)
			}
			if len(oldJWK) > 0 {
				oldJKT, _ = computeJWKThumbprint(oldJWK)
			}
		}
	}

	// Compute new_jkt from the supplied JWK.
	newJKT, err := computeJWKThumbprint(req.NewPublicJWK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_jwk", err.Error()))
		return
	}

	// Persist the new JWK into agent Metadata.
	if agent.Metadata == nil {
		agent.Metadata = map[string]any{}
	}
	agent.Metadata["dpop_public_jwk"] = req.NewPublicJWK
	agent.UpdatedAt = time.Now().UTC()
	if err := s.Store.UpdateAgent(r.Context(), agent); err != nil {
		internal(w, err)
		return
	}

	// Revoke all current tokens for this agent (old key is no longer valid).
	revokedCount, err := s.Store.RevokeOAuthTokensByClientID(r.Context(), agent.ClientID)
	if err != nil {
		// Non-fatal; log but continue — JWK is already updated.
		revokedCount = 0
	}

	// Emit audit event.
	auditMeta := map[string]any{
		"old_jkt":             oldJKT,
		"new_jkt":             newJKT,
		"revoked_token_count": revokedCount,
		"reason":              req.Reason,
	}
	s.auditAgentWithMeta(r, "agent.dpop_key_rotated", agent.ID, auditMeta)

	// Fetch the audit event ID from the last log entry for this agent.
	auditEventID := ""
	if s.AuditLogger != nil {
		logs, qErr := s.AuditLogger.Query(r.Context(), storage.AuditLogQuery{
			TargetID: agent.ID,
			Limit:    1,
		})
		if qErr == nil && len(logs) > 0 {
			auditEventID = logs[0].ID
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"old_jkt":             oldJKT,
		"new_jkt":             newJKT,
		"revoked_token_count": revokedCount,
		"audit_event_id":      auditEventID,
	})
}
