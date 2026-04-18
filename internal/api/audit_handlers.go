package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sharkauth/sharkauth/internal/audit"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// auditListResponse is the response for GET /api/v1/audit-logs.
type auditListResponse struct {
	Data       []*storage.AuditLog `json:"data"`
	NextCursor string              `json:"next_cursor,omitempty"`
	HasMore    bool                `json:"has_more"`
}

// auditExportRequest is the request body for POST /api/v1/audit-logs/export.
type auditExportRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action,omitempty"`
}

// handleListAuditLogs handles GET /api/v1/audit-logs with query filters.
func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 50
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}

	opts := storage.AuditLogQuery{
		Action:   q.Get("action"),
		ActorID:  q.Get("actor_id"),
		TargetID: q.Get("target_id"),
		Status:   q.Get("status"),
		IP:       q.Get("ip"),
		From:     q.Get("from"),
		To:       q.Get("to"),
		Limit:    limit + 1, // fetch one extra to determine has_more
		Cursor:   q.Get("cursor"),
	}

	// Validate date formats if provided
	if opts.From != "" {
		if _, err := time.Parse(time.RFC3339, opts.From); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "invalid_parameter",
				"message": "Invalid 'from' date format, expected RFC3339",
			})
			return
		}
	}
	if opts.To != "" {
		if _, err := time.Parse(time.RFC3339, opts.To); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "invalid_parameter",
				"message": "Invalid 'to' date format, expected RFC3339",
			})
			return
		}
	}

	logs, err := s.AuditLogger.Query(r.Context(), opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to query audit logs",
		})
		return
	}

	hasMore := len(logs) > limit
	if hasMore {
		logs = logs[:limit]
	}

	resp := auditListResponse{
		Data:    logs,
		HasMore: hasMore,
	}

	if resp.Data == nil {
		resp.Data = []*storage.AuditLog{}
	}

	if hasMore && len(logs) > 0 {
		resp.NextCursor = logs[len(logs)-1].ID
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleGetAuditLog handles GET /api/v1/audit-logs/{id}.
func (s *Server) handleGetAuditLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_parameter",
			"message": "Missing audit log ID",
		})
		return
	}

	entry, err := s.AuditLogger.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Audit log entry not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// handleUserAuditLogs handles GET /api/v1/users/{id}/audit-logs.
// Returns logs where the user is either actor OR target.
func (s *Server) handleUserAuditLogs(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_parameter",
			"message": "Missing user ID",
		})
		return
	}

	q := r.URL.Query()

	limit := 50
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 200 {
		limit = 200
	}

	// Query logs where user is actor
	actorOpts := storage.AuditLogQuery{
		ActorID: userID,
		Limit:   limit,
		Cursor:  q.Get("cursor"),
	}
	actorLogs, err := s.AuditLogger.Query(r.Context(), actorOpts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to query audit logs",
		})
		return
	}

	// Query logs where user is target
	targetOpts := storage.AuditLogQuery{
		TargetID: userID,
		Limit:    limit,
		Cursor:   q.Get("cursor"),
	}
	targetLogs, err := s.AuditLogger.Query(r.Context(), targetOpts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to query audit logs",
		})
		return
	}

	// Merge and deduplicate (some logs may have the same user as both actor and target)
	seen := make(map[string]bool)
	var merged []*storage.AuditLog
	for _, l := range actorLogs {
		if !seen[l.ID] {
			seen[l.ID] = true
			merged = append(merged, l)
		}
	}
	for _, l := range targetLogs {
		if !seen[l.ID] {
			seen[l.ID] = true
			merged = append(merged, l)
		}
	}

	// Sort merged results by created_at DESC (both sub-lists are already sorted)
	sortAuditLogsByCreatedAtDesc(merged)

	// Apply limit
	if len(merged) > limit {
		merged = merged[:limit]
	}

	if merged == nil {
		merged = []*storage.AuditLog{}
	}

	resp := auditListResponse{
		Data:    merged,
		HasMore: len(actorLogs) >= limit || len(targetLogs) >= limit,
	}

	if resp.HasMore && len(merged) > 0 {
		resp.NextCursor = merged[len(merged)-1].ID
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleExportAuditLogs handles POST /api/v1/audit-logs/export.
// Exports audit logs as JSON for a required date range.
func (s *Server) handleExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	var req auditExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.From == "" || req.To == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Both 'from' and 'to' date fields are required",
		})
		return
	}

	if _, err := time.Parse(time.RFC3339, req.From); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_parameter",
			"message": "Invalid 'from' date format, expected RFC3339",
		})
		return
	}
	if _, err := time.Parse(time.RFC3339, req.To); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_parameter",
			"message": "Invalid 'to' date format, expected RFC3339",
		})
		return
	}

	// Collect all logs within the date range, paginating through them
	var allLogs []*storage.AuditLog
	cursor := ""
	for {
		opts := storage.AuditLogQuery{
			From:   req.From,
			To:     req.To,
			Action: req.Action,
			Limit:  200,
			Cursor: cursor,
		}

		logs, err := s.AuditLogger.Query(r.Context(), opts)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal_error",
				"message": "Failed to export audit logs",
			})
			return
		}

		allLogs = append(allLogs, logs...)

		if len(logs) < 200 {
			break
		}
		cursor = logs[len(logs)-1].ID
	}

	if allLogs == nil {
		allLogs = []*storage.AuditLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=audit-logs-export.json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{ //#nosec G104 -- write to ResponseWriter; no actionable recovery
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"from":        req.From,
		"to":          req.To,
		"count":       len(allLogs),
		"data":        allLogs,
	})
}

// sortAuditLogsByCreatedAtDesc sorts audit logs by created_at descending.
// Uses a simple insertion sort since the inputs are small (bounded by limit).
func sortAuditLogsByCreatedAtDesc(logs []*storage.AuditLog) {
	for i := 1; i < len(logs); i++ {
		for j := i; j > 0 && logs[j].CreatedAt > logs[j-1].CreatedAt; j-- {
			logs[j], logs[j-1] = logs[j-1], logs[j]
		}
	}
}

// newAuditLogger creates an audit.Logger from the server's store.
func newAuditLogger(store storage.Store) *audit.Logger {
	return audit.NewLogger(store)
}
