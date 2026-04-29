// Package api â€” admin-key-authenticated user management handlers.
//
// Parallels admin_organization_handlers.go: dashboard pages send the admin
// Bearer key, so user-creation flows land here instead of the public
// /auth/signup route. T04 of DASHBOARD_DX_EXECUTION_PLAN.md â€” POST creates a
// new user (password optional; invite-via-magic-link is T05/T06 scope).
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/storage"
)

// adminCreateUserRequest is the POST /admin/users body.
type adminCreateUserRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password,omitempty"`
	Name          string `json:"name,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

// handleAdminCreateUser handles POST /api/v1/admin/users.
// Admin-key auth only (middleware mounted on parent group). Creates a user
// with optional password hashing; when password is omitted the row lands
// with a nil password_hash (T05/T06 wires invite-via-magic-link separately).
func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req adminCreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !emailRegex.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Valid email is required"))
		return
	}

	// Email uniqueness.
	if _, err := s.Store.GetUserByEmail(r.Context(), req.Email); err == nil {
		writeJSON(w, http.StatusConflict, errPayload("email_exists", "An account with this email already exists"))
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		internal(w, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	idSuffix, err := gonanoid.New()
	if err != nil {
		internal(w, err)
		return
	}
	user := &storage.User{
		ID:            "usr_" + idSuffix,
		Email:         req.Email,
		EmailVerified: req.EmailVerified,
		Metadata:      "{}",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if strings.TrimSpace(req.Name) != "" {
		n := strings.TrimSpace(req.Name)
		user.Name = &n
	}
	if req.Password != "" {
		minLen := s.Config.Auth.PasswordMinLength
		if minLen == 0 {
			minLen = 8
		}
		if reason := auth.ValidatePasswordComplexity(req.Password, minLen); reason != "" {
			writeJSON(w, http.StatusBadRequest, errPayload("weak_password", reason))
			return
		}
		hash, err := auth.HashPassword(req.Password, s.Config.Auth.Argon2id)
		if err != nil {
			internal(w, err)
			return
		}
		user.PasswordHash = &hash
		user.HashType = "argon2id"
	}

	if err := s.Store.CreateUser(r.Context(), user); err != nil {
		internal(w, err)
		return
	}

	// Audit log entry â€” mirrors admin_organization_handlers.auditAdminOrg
	// shape but targets the new user. Metadata carries the email for
	// operator-friendly review without needing a user-table join.
	if s.AuditLogger != nil {
		metaBytes, _ := json.Marshal(map[string]any{
			"email":          user.Email,
			"email_verified": user.EmailVerified,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "admin.user.create",
			TargetType: "user",
			TargetID:   user.ID,
			IP:         ipOf(r),
			UserAgent:  uaOf(r),
			Metadata:   string(metaBytes),
			Status:     "success",
		})
	}

	s.emit(r.Context(), storage.WebhookEventUserCreated, userPublic(user))

	writeJSON(w, http.StatusCreated, adminUserToResponse(user))
}
