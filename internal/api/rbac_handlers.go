package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// --- Request/Response types ---

type createRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type roleResponse struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Permissions []*permissionResponse `json:"permissions,omitempty"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
}

type createPermissionRequest struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

type permissionResponse struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	CreatedAt string `json:"created_at"`
}

type attachPermissionRequest struct {
	PermissionID string `json:"permission_id"`
}

type assignRoleRequest struct {
	RoleID string `json:"role_id"`
}

type checkPermissionRequest struct {
	UserID   string `json:"user_id"`
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

type checkPermissionResponse struct {
	Allowed bool `json:"allowed"`
}

// --- Helpers ---

func permToResponse(p *storage.Permission) *permissionResponse {
	return &permissionResponse{
		ID:        p.ID,
		Action:    p.Action,
		Resource:  p.Resource,
		CreatedAt: p.CreatedAt,
	}
}

func roleToResponse(r *storage.Role) *roleResponse {
	return &roleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// --- Role handlers ---

func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
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
			"message": "Role name is required",
		})
		return
	}

	// Check for duplicate name
	_, err := s.Store.GetRoleByName(r.Context(), req.Name)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":   "conflict",
			"message": "A role with this name already exists",
		})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id, _ := gonanoid.New()
	role := &storage.Role{
		ID:          "role_" + id,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.Store.CreateRole(r.Context(), role); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	writeJSON(w, http.StatusCreated, roleToResponse(role))
}

func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.Store.ListRoles(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	result := make([]*roleResponse, len(roles))
	for i, role := range roles {
		result[i] = roleToResponse(role)
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := s.Store.GetRoleByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Role not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	resp := roleToResponse(role)

	// Include permissions
	perms, err := s.Store.GetPermissionsByRoleID(r.Context(), role.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	resp.Permissions = make([]*permissionResponse, len(perms))
	for i, p := range perms {
		resp.Permissions[i] = permToResponse(p)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role, err := s.Store.GetRoleByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Role not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	role.Description = req.Description
	role.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.Store.UpdateRole(r.Context(), role); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, roleToResponse(role))
}

func (s *Server) handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.Store.GetRoleByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Role not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	if err := s.Store.DeleteRole(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Permission handlers ---

func (s *Server) handleCreatePermission(w http.ResponseWriter, r *http.Request) {
	var req createPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.Action == "" || req.Resource == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Action and resource are required",
		})
		return
	}

	// Check for duplicate
	_, err := s.Store.GetPermissionByActionResource(r.Context(), req.Action, req.Resource)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":   "conflict",
			"message": "A permission with this action and resource already exists",
		})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id, _ := gonanoid.New()
	perm := &storage.Permission{
		ID:        "perm_" + id,
		Action:    req.Action,
		Resource:  req.Resource,
		CreatedAt: now,
	}

	if err := s.Store.CreatePermission(r.Context(), perm); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	writeJSON(w, http.StatusCreated, permToResponse(perm))
}

func (s *Server) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := s.Store.ListPermissions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	result := make([]*permissionResponse, len(perms))
	for i, p := range perms {
		result[i] = permToResponse(p)
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Role-Permission handlers ---

func (s *Server) handleAttachPermission(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	_, err := s.Store.GetRoleByID(r.Context(), roleID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Role not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	var req attachPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.PermissionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "permission_id is required",
		})
		return
	}

	_, err = s.Store.GetPermissionByID(r.Context(), req.PermissionID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Permission not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	if err := s.Store.AttachPermissionToRole(r.Context(), roleID, req.PermissionID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDetachPermission(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	permID := chi.URLParam(r, "pid")

	if err := s.Store.DetachPermissionFromRole(r.Context(), roleID, permID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- User-Role handlers ---

func (s *Server) handleAssignRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	_, err := s.Store.GetUserByID(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "User not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	var req assignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.RoleID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "role_id is required",
		})
		return
	}

	_, err = s.Store.GetRoleByID(r.Context(), req.RoleID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Role not found",
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	if err := s.Store.AssignRoleToUser(r.Context(), userID, req.RoleID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	roleID := chi.URLParam(r, "rid")

	if err := s.Store.RemoveRoleFromUser(r.Context(), userID, roleID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	roles, err := s.Store.GetRolesByUserID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	result := make([]*roleResponse, len(roles))
	for i, role := range roles {
		result[i] = roleToResponse(role)
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleListUserPermissions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if s.RBAC == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "RBAC not configured",
		})
		return
	}

	perms, err := s.RBAC.GetEffectivePermissions(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	result := make([]*permissionResponse, len(perms))
	for i, p := range perms {
		result[i] = permToResponse(p)
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Auth check handler ---

func (s *Server) handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	var req checkPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.UserID == "" || req.Action == "" || req.Resource == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "user_id, action, and resource are required",
		})
		return
	}

	if s.RBAC == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "RBAC not configured",
		})
		return
	}

	allowed, err := s.RBAC.HasPermission(r.Context(), req.UserID, req.Action, req.Resource)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Permission check failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, checkPermissionResponse{Allowed: allowed})
}
