package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// adminDeviceCodeResponse is the wire shape for the admin device-code queue.
// Mirrors OAuthDeviceCode but resolves the agent display name and trims the
// internal hash field (already json:"-" on the storage struct).
type adminDeviceCodeResponse struct {
	UserCode     string     `json:"user_code"`
	ClientID     string     `json:"client_id"`
	AgentName    string     `json:"agent_name,omitempty"`
	Scope        string     `json:"scope"`
	Resource     string     `json:"resource,omitempty"`
	UserID       string     `json:"user_id,omitempty"`
	Status       string     `json:"status"`
	LastPolledAt *time.Time `json:"last_polled_at,omitempty"`
	PollInterval int        `json:"poll_interval"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// handleAdminListDeviceCodes handles GET /api/v1/admin/oauth/device-codes.
// Returns pending (status=pending, not yet expired) device codes by default;
// when ?status=all is passed, returns every row regardless of state. Admin-
// scope queue used by the dashboard's device-flow approval surface.
func (s *Server) handleAdminListDeviceCodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	codes, err := s.Store.ListPendingDeviceCodes(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	resp := make([]adminDeviceCodeResponse, 0, len(codes))
	agentCache := make(map[string]string)
	for _, dc := range codes {
		row := adminDeviceCodeResponse{
			UserCode:     dc.UserCode,
			ClientID:     dc.ClientID,
			Scope:        dc.Scope,
			Resource:     dc.Resource,
			UserID:       dc.UserID,
			Status:       dc.Status,
			LastPolledAt: dc.LastPolledAt,
			PollInterval: dc.PollInterval,
			ExpiresAt:    dc.ExpiresAt,
			CreatedAt:    dc.CreatedAt,
		}
		if name, ok := agentCache[dc.ClientID]; ok {
			row.AgentName = name
		} else if agent, err := s.Store.GetAgentByClientID(ctx, dc.ClientID); err == nil {
			row.AgentName = agent.Name
			agentCache[dc.ClientID] = agent.Name
		}
		resp = append(resp, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": resp, "total": len(resp)})
}

// handleAdminApproveDeviceCode handles POST /api/v1/admin/oauth/device-codes/{user_code}/approve.
// Admin override of the user-facing approval flow — flips status to "approved"
// without requiring a session for the user_id field. The body may carry
// {"user_id": "usr_…"} so the resulting access token is bound to a specific
// user; if omitted the existing user_id (set by an earlier verify step) stays.
func (s *Server) handleAdminApproveDeviceCode(w http.ResponseWriter, r *http.Request) {
	s.adminDecideDeviceCode(w, r, "approved")
}

// handleAdminDenyDeviceCode handles POST /api/v1/admin/oauth/device-codes/{user_code}/deny.
func (s *Server) handleAdminDenyDeviceCode(w http.ResponseWriter, r *http.Request) {
	s.adminDecideDeviceCode(w, r, "denied")
}

func (s *Server) adminDecideDeviceCode(w http.ResponseWriter, r *http.Request, decision string) {
	ctx := r.Context()
	userCode := chi.URLParam(r, "user_code")

	dc, err := s.Store.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Device code not found"))
		return
	}
	if time.Now().After(dc.ExpiresAt) {
		writeJSON(w, http.StatusGone, errPayload("expired", "Device code has expired"))
		return
	}
	if dc.Status != "pending" {
		writeJSON(w, http.StatusConflict, errPayload("already_decided", "Device code is no longer pending"))
		return
	}

	// Optional body: {user_id} to bind the token to a specific user.
	var req struct {
		UserID string `json:"user_id,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	uid := req.UserID
	if uid == "" {
		uid = dc.UserID
	}

	if err := s.Store.UpdateDeviceCodeStatus(ctx, dc.DeviceCodeHash, decision, uid); err != nil {
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		meta, _ := json.Marshal(map[string]any{
			"user_code": userCode,
			"client_id": dc.ClientID,
			"user_id":   uid,
			"decision":  decision,
		})
		_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
			ActorType:  "admin",
			Action:     "oauth.device." + decision,
			TargetType: "device_code",
			TargetID:   userCode,
			IP:         ipOf(r),
			UserAgent:  uaOf(r),
			Metadata:   string(meta),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_code": userCode,
		"status":    decision,
	})
}
