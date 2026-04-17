package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/storage"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// --- Request/Response types for org RBAC ---

type orgRoleResponse struct {
	ID          string `json:"id"`
	OrgID       string `json:"organization_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsBuiltin   bool   `json:"is_builtin"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func orgRoleToResponse(r *storage.OrgRole) orgRoleResponse {
	return orgRoleResponse{
		ID:          r.ID,
		OrgID:       r.OrganizationID,
		Name:        r.Name,
		Description: r.Description,
		IsBuiltin:   r.IsBuiltin,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type createOrgRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateOrgRoleRequest struct {
	Name              *string        `json:"name,omitempty"`
	Description       *string        `json:"description,omitempty"`
	AttachPermissions []orgPermInput `json:"attach_permissions,omitempty"`
	DetachPermissions []orgPermInput `json:"detach_permissions,omitempty"`
}

type orgPermInput struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

type orgPermResponse struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

type orgRoleWithPermsResponse struct {
	orgRoleResponse
	Permissions []orgPermResponse `json:"permissions"`
}

// auditOrgRBAC logs an org RBAC event.
func (s *Server) auditOrgRBAC(ctx context.Context, actor, action, orgID, targetID, ip, ua string) {
	if s.AuditLogger == nil {
		return
	}
	_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
		ActorID:    actor,
		ActorType:  "user",
		Action:     action,
		TargetType: "org_role",
		TargetID:   targetID,
		IP:         ip,
		UserAgent:  ua,
		Status:     "success",
	})
}

// --- Handlers ---

// GET /organizations/{org_id}/roles
func (s *Server) handleListOrgRoles(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	roles, err := s.Store.GetOrgRolesByOrgID(r.Context(), orgID)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]orgRoleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, orgRoleToResponse(role))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

// POST /organizations/{org_id}/roles
func (s *Server) handleCreateOrgRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	actor := mw.GetUserID(r.Context())

	var req createOrgRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Name is required"))
		return
	}

	role, err := s.RBAC.CreateOrgRole(r.Context(), orgID, req.Name, req.Description)
	if err != nil {
		internal(w, err)
		return
	}

	s.auditOrgRBAC(r.Context(), actor, "org.role.create", orgID, role.ID, ipOf(r), uaOf(r))
	writeJSON(w, http.StatusCreated, orgRoleToResponse(role))
}

// GET /organizations/{org_id}/roles/{role_id}
func (s *Server) handleGetOrgRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "role_id")
	role, err := s.Store.GetOrgRoleByID(r.Context(), roleID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Role not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	perms, err := s.Store.GetOrgRolePermissions(r.Context(), roleID)
	if err != nil {
		internal(w, err)
		return
	}
	permOut := make([]orgPermResponse, 0, len(perms))
	for _, p := range perms {
		permOut = append(permOut, orgPermResponse{Action: p.Action, Resource: p.Resource})
	}
	writeJSON(w, http.StatusOK, orgRoleWithPermsResponse{orgRoleToResponse(role), permOut})
}

// PATCH /organizations/{org_id}/roles/{role_id}
func (s *Server) handleUpdateOrgRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	roleID := chi.URLParam(r, "role_id")
	actor := mw.GetUserID(r.Context())

	role, err := s.Store.GetOrgRoleByID(r.Context(), roleID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Role not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	var req updateOrgRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	name := role.Name
	desc := role.Description
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}
	if req.Description != nil {
		desc = *req.Description
	}

	if err := s.Store.UpdateOrgRole(r.Context(), roleID, name, desc); err != nil {
		internal(w, err)
		return
	}

	// Attach permissions
	for _, p := range req.AttachPermissions {
		if err := s.RBAC.AttachOrgPermission(r.Context(), roleID, p.Action, p.Resource); err != nil {
			internal(w, err)
			return
		}
		s.auditOrgRBAC(r.Context(), actor, "org.permission.attach", orgID, roleID, ipOf(r), uaOf(r))
	}

	// Detach permissions
	for _, p := range req.DetachPermissions {
		if err := s.RBAC.DetachOrgPermission(r.Context(), roleID, p.Action, p.Resource); err != nil {
			internal(w, err)
			return
		}
		s.auditOrgRBAC(r.Context(), actor, "org.permission.detach", orgID, roleID, ipOf(r), uaOf(r))
	}

	updated, err := s.Store.GetOrgRoleByID(r.Context(), roleID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orgRoleToResponse(updated))
}

// DELETE /organizations/{org_id}/roles/{role_id}
func (s *Server) handleDeleteOrgRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	roleID := chi.URLParam(r, "role_id")
	actor := mw.GetUserID(r.Context())

	err := s.RBAC.DeleteOrgRole(r.Context(), orgID, roleID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Role not found"))
		return
	}
	if errors.Is(err, rbac.ErrBuiltinRole) {
		writeJSON(w, http.StatusConflict, errPayload("builtin_role", "Cannot delete a builtin role"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	s.auditOrgRBAC(r.Context(), actor, "org.role.delete", orgID, roleID, ipOf(r), uaOf(r))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Role deleted"})
}

// POST /organizations/{org_id}/members/{user_id}/roles/{role_id}
func (s *Server) handleGrantOrgRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "user_id")
	roleID := chi.URLParam(r, "role_id")
	actor := mw.GetUserID(r.Context())

	if err := s.RBAC.GrantOrgRole(r.Context(), orgID, targetUserID, roleID, actor); err != nil {
		internal(w, err)
		return
	}

	s.auditOrgRBAC(r.Context(), actor, "org.role.grant", orgID, roleID, ipOf(r), uaOf(r))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Role granted"})
}

// DELETE /organizations/{org_id}/members/{user_id}/roles/{role_id}
func (s *Server) handleRevokeOrgRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "user_id")
	roleID := chi.URLParam(r, "role_id")
	actor := mw.GetUserID(r.Context())

	if err := s.RBAC.RevokeOrgRole(r.Context(), orgID, targetUserID, roleID); err != nil {
		internal(w, err)
		return
	}

	s.auditOrgRBAC(r.Context(), actor, "org.role.revoke", orgID, roleID, ipOf(r), uaOf(r))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Role revoked"})
}

// GET /organizations/{org_id}/members/{user_id}/permissions
func (s *Server) handleGetEffectiveOrgPerms(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "user_id")

	perms, err := s.RBAC.GetEffectiveOrgPermissions(r.Context(), targetUserID, orgID)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]orgPermResponse, 0, len(perms))
	for _, p := range perms {
		out = append(out, orgPermResponse{Action: p.Action, Resource: p.Resource})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}
