package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	mw "github.com/shark-auth/shark/internal/api/middleware"
	"github.com/shark-auth/shark/internal/storage"
)

// sessionResponse is the wire representation of a session. Hides IP on
// self-service endpoints by default â€” admin response embeds this + extras.
type sessionResponse struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	IP             string `json:"ip,omitempty"`
	UserAgent      string `json:"user_agent,omitempty"`
	MFAPassed      bool   `json:"mfa_passed"`
	AuthMethod     string `json:"auth_method"`
	ExpiresAt      string `json:"expires_at"`
	CreatedAt      string `json:"created_at"`
	LastActivityAt string `json:"last_activity_at,omitempty"`
	Current        bool   `json:"current,omitempty"`
}

type adminSessionResponse struct {
	sessionResponse
	UserEmail       string `json:"user_email"`
	UserMFAEnabled  bool   `json:"user_mfa_enabled"`
	UserMFAVerified bool   `json:"user_mfa_verified"`
	JTI             string `json:"jti,omitempty"`
}

type sessionListResponse struct {
	Data       []sessionResponse `json:"data"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type adminSessionListResponse struct {
	Data       []adminSessionResponse `json:"data"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

// --- Self-service /auth/sessions ---

func (s *Server) handleListMySessions(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	currentID := mw.GetSessionID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized", "message": "No valid session"})
		return
	}

	sessions, err := s.Store.GetSessionsByUserID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}

	out := make([]sessionResponse, 0, len(sessions))
	for _, se := range sessions {
		out = append(out, sessionResponse{
			ID:             se.ID,
			UserID:         se.UserID,
			IP:             se.IP,
			UserAgent:      se.UserAgent,
			MFAPassed:      se.MFAPassed,
			AuthMethod:     se.AuthMethod,
			ExpiresAt:      se.ExpiresAt,
			CreatedAt:      se.CreatedAt,
			LastActivityAt: se.CreatedAt,
			Current:        se.ID == currentID,
		})
	}
	writeJSON(w, http.StatusOK, sessionListResponse{Data: out})
}

func (s *Server) handleRevokeMySession(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	sessID := chi.URLParam(r, "id")

	sess, err := s.Store.GetSessionByID(r.Context(), sessID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Session not found"})
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	// Users can only revoke their own sessions.
	if sess.UserID != userID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Session not found"})
		return
	}

	if err := s.Store.DeleteSession(r.Context(), sessID); err != nil {
		internal(w, err)
		return
	}
	s.evictSessionAuth(sessID)
	s.auditSessionRevoke(r.Context(), "user", userID, userID, sessID, ipOf(r), uaOf(r))
	s.emit(r.Context(), storage.WebhookEventSessionRevoked, map[string]string{
		"session_id": sessID, "user_id": userID, "revoked_by": "user",
	})

	w.WriteHeader(http.StatusNoContent)
}

// --- Admin /admin/sessions ---

func (s *Server) handleAdminListSessions(w http.ResponseWriter, r *http.Request) {
	opts := storage.ListSessionsOpts{
		UserID:     r.URL.Query().Get("user_id"),
		AuthMethod: r.URL.Query().Get("auth_method"),
		Cursor:     r.URL.Query().Get("cursor"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opts.Limit = n
		}
	}
	if v := r.URL.Query().Get("mfa_passed"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "invalid_request", "message": "mfa_passed must be boolean",
			})
			return
		}
		opts.MFAPassed = &b
	}

	rows, err := s.Store.ListActiveSessions(r.Context(), opts)
	if err != nil {
		internal(w, err)
		return
	}

	data := make([]adminSessionResponse, 0, len(rows))
	for _, sw := range rows {
		data = append(data, adminSessionResponse{
			sessionResponse: sessionResponse{
				ID:             sw.ID,
				UserID:         sw.UserID,
				IP:             sw.IP,
				UserAgent:      sw.UserAgent,
				MFAPassed:      sw.MFAPassed,
				AuthMethod:     sw.AuthMethod,
				ExpiresAt:      sw.ExpiresAt,
				CreatedAt:      sw.CreatedAt,
				LastActivityAt: sw.CreatedAt,
			},
			UserEmail:       sw.UserEmail,
			UserMFAEnabled:  sw.UserMFAEnabled,
			UserMFAVerified: sw.UserMFAVerified,
		})
	}

	// Keyset cursor: emit only when the page was full (more likely rows after).
	// Cursor encodes the last tuple so the caller can pass it as-is.
	var next string
	if len(rows) > 0 && (opts.Limit == 0 || len(rows) >= effectiveLimit(opts.Limit)) {
		last := rows[len(rows)-1]
		next = last.CreatedAt + "|" + last.ID
	}
	writeJSON(w, http.StatusOK, adminSessionListResponse{Data: data, NextCursor: next})
}

func (s *Server) handleAdminDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessID := chi.URLParam(r, "id")
	actor := actorID(r.Context())

	sess, err := s.Store.GetSessionByID(r.Context(), sessID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Session not found"})
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	if err := s.Store.DeleteSession(r.Context(), sessID); err != nil {
		internal(w, err)
		return
	}
	s.evictSessionAuth(sessID)
	s.auditSessionRevoke(r.Context(), "admin", actor, sess.UserID, sessID, ipOf(r), uaOf(r))
	s.emit(r.Context(), storage.WebhookEventSessionRevoked, map[string]string{
		"session_id": sessID, "user_id": sess.UserID, "revoked_by": "admin",
	})

	w.WriteHeader(http.StatusNoContent)
}

// --- Per-user session endpoints (admin scope) ---

func (s *Server) handleListUserSessions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	sessions, err := s.Store.GetSessionsByUserID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]sessionResponse, 0, len(sessions))
	for _, se := range sessions {
		out = append(out, sessionResponse{
			ID:             se.ID,
			UserID:         se.UserID,
			IP:             se.IP,
			UserAgent:      se.UserAgent,
			MFAPassed:      se.MFAPassed,
			AuthMethod:     se.AuthMethod,
			ExpiresAt:      se.ExpiresAt,
			CreatedAt:      se.CreatedAt,
			LastActivityAt: se.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, sessionListResponse{Data: out})
}

func (s *Server) handleRevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	actor := actorID(r.Context())

	ids, err := s.Store.DeleteSessionsByUserID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}

	// Granular audit + emission: one entry per revoked session so compliance
	// review and downstream consumers can reconstruct exactly which device
	// tokens were invalidated.
	ip, ua := ipOf(r), uaOf(r)
	for _, id := range ids {
		s.evictSessionAuth(id)
		s.auditSessionRevoke(r.Context(), "admin", actor, userID, id, ip, ua)
		s.emit(r.Context(), storage.WebhookEventSessionRevoked, map[string]string{
			"session_id": id, "user_id": userID, "revoked_by": "admin",
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Sessions revoked",
		"count":   len(ids),
	})
}

// handleAdminRevokeAllSessions handles DELETE /api/v1/admin/sessions.
// Revokes (deletes) every active session and returns {"revoked": N}.
func (s *Server) handleAdminRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	actor := actorID(r.Context())

	count, err := s.Store.DeleteAllActiveSessions(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	if s.AuthCache != nil {
		s.AuthCache.Clear()
	}

	// Single audit entry summarising the bulk action.
	s.auditSessionRevoke(r.Context(), "admin", actor, "all", "all", ipOf(r), uaOf(r))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"revoked": count,
	})
}

// --- Helpers ---

func (s *Server) auditSessionRevoke(ctx context.Context, actorType, actorID, targetUserID, sessionID, ip, ua string) {
	if s.AuditLogger == nil {
		return
	}
	meta, _ := json.Marshal(map[string]string{
		"session_id":     sessionID,
		"target_user_id": targetUserID,
	})
	_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
		ActorID:    actorID,
		ActorType:  actorType,
		Action:     "session.revoke",
		TargetType: "session",
		TargetID:   sessionID,
		IP:         ip,
		UserAgent:  ua,
		Metadata:   string(meta),
		Status:     "success",
	})
}

func (s *Server) evictSessionAuth(sessionID string) {
	if s.AuthCache != nil && sessionID != "" {
		s.AuthCache.Delete("session:" + sessionID)
	}
}

// handlePurgeExpiredSessions handles POST /api/v1/admin/sessions/purge-expired.
// Deletes all sessions whose expires_at is in the past and returns the count deleted.
func (s *Server) handlePurgeExpiredSessions(w http.ResponseWriter, r *http.Request) {
	count, err := s.Store.DeleteExpiredSessions(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": count})
}

// effectiveLimit mirrors the clamp applied inside ListActiveSessions so the
// handler can decide whether a "next page probably exists" signal is warranted.
func effectiveLimit(n int) int {
	if n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}

func ipOf(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		return v
	}
	return r.RemoteAddr
}

func uaOf(r *http.Request) string { return r.Header.Get("User-Agent") }

// actorID extracts the acting API key ID (admin calls) from context, falling
// back to user ID for session-authenticated paths.
func actorID(ctx context.Context) string {
	if v, ok := ctx.Value(mw.APIKeyIDKey).(string); ok && v != "" {
		return v
	}
	return mw.GetUserID(ctx)
}
