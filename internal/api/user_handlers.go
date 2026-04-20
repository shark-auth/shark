package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// adminUserResponse is the admin view of a user (includes metadata).
type adminUserResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"emailVerified"`
	Name          *string `json:"name,omitempty"`
	AvatarURL     *string `json:"avatarUrl,omitempty"`
	MFAEnabled    bool    `json:"mfaEnabled"`
	Metadata      string  `json:"metadata"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	LastLoginAt   *string `json:"last_login_at,omitempty"`
}

func adminUserToResponse(u *storage.User) adminUserResponse {
	return adminUserResponse{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Name:          u.Name,
		AvatarURL:     u.AvatarURL,
		MFAEnabled:    u.MFAEnabled,
		Metadata:      u.Metadata,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		LastLoginAt:   u.LastLoginAt,
	}
}

// handleListUsers handles GET /api/v1/users
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Accept both limit/offset and dashboard's page/per_page.
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if perPage, _ := strconv.Atoi(q.Get("per_page")); perPage > 0 {
		limit = perPage
	}
	if page, _ := strconv.Atoi(q.Get("page")); page > 0 && limit > 0 {
		offset = (page - 1) * limit
	}
	if limit <= 0 {
		limit = 50
	}

	opts := storage.ListUsersOpts{
		Limit:      limit,
		Offset:     offset,
		Search:     q.Get("search"),
		RoleID:     q.Get("role_id"),
		AuthMethod: q.Get("auth_method"),
		OrgID:      q.Get("org_id"),
	}
	if v := q.Get("mfa_enabled"); v == "true" || v == "false" {
		b := v == "true"
		opts.MFAEnabled = &b
	}
	if v := q.Get("email_verified"); v == "true" || v == "false" {
		b := v == "true"
		opts.EmailVerified = &b
	}

	users, err := s.Store.ListUsers(r.Context(), opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Total count uses the same filters but ignores limit/offset.
	totalOpts := opts
	totalOpts.Limit = 1000000
	totalOpts.Offset = 0
	allUsers, err := s.Store.ListUsers(r.Context(), totalOpts)
	total := 0
	if err == nil {
		total = len(allUsers)
	}

	resp := make([]adminUserResponse, len(users))
	for i, u := range users {
		resp[i] = adminUserToResponse(u)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"users": resp,
		"total": total,
	})
}

// handleGetUser handles GET /api/v1/users/{id}
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := s.Store.GetUserByID(r.Context(), id)
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

	writeJSON(w, http.StatusOK, adminUserToResponse(user))
}

// handleDeleteUser handles DELETE /api/v1/users/{id}
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Verify user exists
	_, err := s.Store.GetUserByID(r.Context(), id)
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

	// DeleteUser cascades via ON DELETE CASCADE in the schema
	if err := s.Store.DeleteUser(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	s.emit(r.Context(), storage.WebhookEventUserDeleted, map[string]string{"id": id})

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "User deleted",
	})
}

// updateUserRequest is the request body for PATCH /api/v1/users/{id}
type updateUserRequest struct {
	Email         *string `json:"email,omitempty"`
	Name          *string `json:"name,omitempty"`
	EmailVerified *bool   `json:"email_verified,omitempty"`
	Metadata      *string `json:"metadata,omitempty"`
}

// handleUpdateUser handles PATCH /api/v1/users/{id}
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	user, err := s.Store.GetUserByID(r.Context(), id)
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

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Name != nil {
		user.Name = req.Name
	}
	if req.EmailVerified != nil {
		user.EmailVerified = *req.EmailVerified
	}
	if req.Metadata != nil {
		user.Metadata = *req.Metadata
	}
	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, adminUserToResponse(user))
}

// handleListUserOAuthAccounts handles GET /api/v1/users/{id}/oauth-accounts.
// Returns all OAuth provider accounts linked to the user.
func (s *Server) handleListUserOAuthAccounts(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	accounts, err := s.Store.GetOAuthAccountsByUserID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	if accounts == nil {
		accounts = []*storage.OAuthAccount{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"oauth_accounts": accounts,
	})
}

// handleDeleteUserOAuthAccount handles DELETE /api/v1/users/{id}/oauth-accounts/{oauthId}.
// Unlinks a specific OAuth provider account from the user. Returns 204 on success.
func (s *Server) handleDeleteUserOAuthAccount(w http.ResponseWriter, r *http.Request) {
	oauthID := chi.URLParam(r, "oauthId")

	if err := s.Store.DeleteOAuthAccount(r.Context(), oauthID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error":   "not_found",
				"message": "OAuth account not found",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteMe handles DELETE /api/v1/auth/me (user self-deletion)
func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "No valid session",
		})
		return
	}

	// Verify user exists
	_, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "User not found",
		})
		return
	}

	// Delete the user (cascades via ON DELETE CASCADE)
	if err := s.Store.DeleteUser(r.Context(), userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	s.emit(r.Context(), storage.WebhookEventUserDeleted, map[string]string{"id": userID})

	// Clear session cookie
	s.SessionManager.ClearSessionCookie(w)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Account deleted",
	})
}
