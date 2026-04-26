package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/auth"
	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// emailRegex is a simple regex for email validation.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// signupRequest is the request body for POST /api/v1/auth/signup.
type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// loginRequest is the request body for POST /api/v1/auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// passwordResetSendRequest is the request body for POST /api/v1/auth/password/send-reset-link.
type passwordResetSendRequest struct {
	Email string `json:"email"`
}

// passwordReset is the request body for POST /api/v1/auth/password/reset.
type passwordReset struct {
	Token string `json:"token"`
	Password string `json:"password"`
}

// changePasswordRequest is the request body for POST /api/v1/auth/password/change.
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// userResponse is the JSON response for user data.
type userResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"emailVerified"`
	Name          *string `json:"name,omitempty"`
	AvatarURL     *string `json:"avatarUrl,omitempty"`
	MFAEnabled    bool    `json:"mfaEnabled"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

func userToResponse(u *storage.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Name:          u.Name,
		AvatarURL:     u.AvatarURL,
		MFAEnabled:    u.MFAEnabled,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	// Validate email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !emailRegex.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_email",
			"message": "Invalid email address",
		})
		return
	}

	// Validate password complexity
	minLen := s.Config.Auth.PasswordMinLength
	if minLen == 0 {
		minLen = 8
	}
	if reason := auth.ValidatePasswordComplexity(req.Password, minLen); reason != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "weak_password",
			"message": reason,
		})
		return
	}

	// Check if email already exists
	_, err := s.Store.GetUserByEmail(r.Context(), req.Email)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":   "email_taken",
			"message": "An account with this email already exists",
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

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password, s.Config.Auth.Argon2id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Create user
	now := time.Now().UTC().Format(time.RFC3339)
	id, _ := gonanoid.New()
	user := &storage.User{
		ID:           "usr_" + id,
		Email:        req.Email,
		PasswordHash: &passwordHash,
		HashType:     "argon2id",
		Metadata:     "{}",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.Name != "" {
		user.Name = &req.Name
	}

	if err := s.Store.CreateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Phase 6 F3: fire auth flow hook. Runs AFTER the user row lands so
	// block/redirect outcomes leave the account in place but withhold the
	// session — matches the documented "user created but login gated"
	// semantics admins rely on.
	if s.runAuthFlow(w, r, storage.AuthFlowTriggerSignup, user, req.Password) {
		return
	}

	// Create session
	sess, err := s.SessionManager.CreateSession(r.Context(), user.ID, r.RemoteAddr, r.UserAgent(), "password")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Set session cookie
	s.SessionManager.SetSessionCookie(w, sess.ID)

	s.emit(r.Context(), storage.WebhookEventUserCreated, userPublic(user))

	resp := map[string]interface{}{}
	for k, v := range userResponseMap(userToResponse(user)) {
		resp[k] = v
	}
	if s.JWTManager != nil && s.Config.Auth.JWT.Enabled {
		if s.Config.Auth.JWT.Mode == "access_refresh" {
			access, refresh, err := s.JWTManager.IssueAccessRefreshPair(r.Context(), user, sess.ID, true)
			if err == nil {
				resp["access_token"] = access
				resp["refresh_token"] = refresh
			}
		} else {
			token, err := s.JWTManager.IssueSessionJWT(r.Context(), user, sess.ID, true)
			if err == nil {
				resp["token"] = token
			}
		}
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Check account lockout
	if s.LockoutManager.IsLocked(req.Email) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error":   "account_locked",
			"message": "Too many failed attempts. Please try again later.",
		})
		return
	}

	// Find user by email (don't leak whether the email exists)
	user, err := s.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		s.LockoutManager.RecordFailure(req.Email)
		s.recordLoginFailure(r, "", req.Email)
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_credentials",
			"message": "Invalid email or password",
		})
		return
	}

	// User must have a password
	if user.PasswordHash == nil {
		s.LockoutManager.RecordFailure(req.Email)
		s.recordLoginFailure(r, user.ID, req.Email)
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_credentials",
			"message": "Invalid email or password",
		})
		return
	}

	// Verify password
	match, err := auth.VerifyPassword(req.Password, *user.PasswordHash)
	if err != nil || !match {
		s.LockoutManager.RecordFailure(req.Email)
		s.recordLoginFailure(r, user.ID, req.Email)
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_credentials",
			"message": "Invalid email or password",
		})
		return
	}

	// Clear lockout on successful login
	s.LockoutManager.RecordSuccess(req.Email)

	// Update last_login_at
	now := time.Now().UTC().Format(time.RFC3339)
	user.LastLoginAt = &now
	_ = s.Store.UpdateUser(r.Context(), user)

	// If password needs rehash (e.g. bcrypt from Auth0 migration), rehash to argon2id
	if auth.NeedsRehash(*user.PasswordHash) {
		newHash, err := auth.HashPassword(req.Password, s.Config.Auth.Argon2id)
		if err == nil {
			user.PasswordHash = &newHash
			user.HashType = "argon2id"
			user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			_ = s.Store.UpdateUser(r.Context(), user)
		}
	}

	// Phase 6 F3: fire auth flow hook. Runs after password (+ lockout) is
	// cleared but before a session cookie is minted so block/redirect
	// outcomes don't leak auth state to the client.
	if s.runAuthFlow(w, r, storage.AuthFlowTriggerLogin, user, "") {
		return
	}

	// Check if MFA is enabled
	mfaPassed := true
	if user.MFAEnabled {
		mfaPassed = false
	}

	// Create session
	sess, err := s.SessionManager.CreateSessionWithMFA(r.Context(), user.ID, r.RemoteAddr, r.UserAgent(), "password", mfaPassed)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Set session cookie
	s.SessionManager.SetSessionCookie(w, sess.ID)

	if user.MFAEnabled {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"mfaRequired": true,
		})
		return
	}

	// Issue JWT alongside cookie if enabled (§1.4).
	resp := map[string]interface{}{}
	for k, v := range userResponseMap(userToResponse(user)) {
		resp[k] = v
	}
	if s.JWTManager != nil && s.Config.Auth.JWT.Enabled {
		if s.Config.Auth.JWT.Mode == "access_refresh" {
			access, refresh, err := s.JWTManager.IssueAccessRefreshPair(r.Context(), user, sess.ID, mfaPassed)
			if err == nil {
				resp["access_token"] = access
				resp["refresh_token"] = refresh
			}
		} else {
			token, err := s.JWTManager.IssueSessionJWT(r.Context(), user, sess.ID, mfaPassed)
			if err == nil {
				resp["token"] = token
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// If the request used a JWT, revoke its JTI.
	if mw.GetAuthMethod(ctx) == "jwt" {
		if claims := mw.GetClaims(ctx); claims != nil && s.JWTManager != nil {
			_ = s.JWTManager.RevokeJTI(ctx, claims.ID, claims.ExpiresAt.Time)
		}
	}

	// Delete the session (cookie path).
	sessionID, err := s.SessionManager.GetSessionFromRequest(r)
	if err == nil && sessionID != "" {
		_ = s.Store.DeleteSession(ctx, sessionID)
	}

	s.SessionManager.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "No valid session",
		})
		return
	}

	user, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "User not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (s *Server) handlePasswordResetSend(w http.ResponseWriter, r *http.Request) {
	var req passwordResetSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !emailRegex.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_email",
			"message": "Invalid email address",
		})
		return
	}

	successMsg := map[string]string{
		"message": "If an account with that email exists, a password reset link has been sent",
	}

	// Always return 200 to avoid leaking whether the email exists
	if s.MagicLinkManager == nil {
		writeJSON(w, http.StatusOK, successMsg)
		return
	}

	_, err := s.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeJSON(w, http.StatusOK, successMsg)
		return
	}

	// Send password reset email (always return 200 to avoid leaking info about email existence)
	if err := s.MagicLinkManager.SendPasswordReset(r.Context(), req.Email); err != nil {
		slog.Error("failed to send password reset email", "email", req.Email, "error", err)
	}

	writeJSON(w, http.StatusOK, successMsg)
}

func (s *Server) handlePasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordReset
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.Token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Token is required",
		})
		return
	}

	minLen := s.Config.Auth.PasswordMinLength
	if minLen == 0 {
		minLen = 8
	}
	if reason := auth.ValidatePasswordComplexity(req.Password, minLen); reason != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "weak_password",
			"message": reason,
		})
		return
	}

	// Verify token and get associated email
	tokenEmail, err := s.MagicLinkManager.VerifyPasswordResetToken(r.Context(), req.Token)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_token",
			"message": "Invalid or expired reset token",
		})
		return
	}

	user, err := s.Store.GetUserByEmail(r.Context(), tokenEmail)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_token",
			"message": "Invalid or expired reset token",
		})
		return
	}

	passwordHash, err := auth.HashPassword(req.Password, s.Config.Auth.Argon2id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	user.PasswordHash = &passwordHash
	user.HashType = "argon2id"
	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Phase 6 F3: fire auth flow hook. The password has already rotated —
	// block/redirect at this point gates the confirmation response only,
	// which is the documented trade-off for "flow runs after the mutation".
	if s.runAuthFlow(w, r, storage.AuthFlowTriggerPasswordReset, user, req.Password) {
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "No valid session",
		})
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	minLen := s.Config.Auth.PasswordMinLength
	if minLen == 0 {
		minLen = 8
	}
	if reason := auth.ValidatePasswordComplexity(req.NewPassword, minLen); reason != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "weak_password",
			"message": reason,
		})
		return
	}

	user, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "User not found",
		})
		return
	}

	// If user already has a password, verify the current one
	if user.PasswordHash != nil {
		if req.CurrentPassword == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "current_password_required",
				"message": "Current password is required",
			})
			return
		}
		match, err := auth.VerifyPassword(req.CurrentPassword, *user.PasswordHash)
		if err != nil || !match {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error":   "invalid_credentials",
				"message": "Current password is incorrect",
			})
			return
		}
	}

	passwordHash, err := auth.HashPassword(req.NewPassword, s.Config.Auth.Argon2id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	user.PasswordHash = &passwordHash
	user.HashType = "argon2id"
	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Password changed successfully",
	})
}

// recordLoginFailure emits a `user.login` audit row with status=failure so the
// admin stats counter (failed_logins_24h) and the audit page reflect real
// activity. actorID may be empty when the email doesn't match a user.
// failureReason is a short tag (e.g. "user_not_found", "no_password",
// "bad_password") that helps operators triage spikes without needing to
// correlate against application logs.
func (s *Server) recordLoginFailure(r *http.Request, actorID, email string) {
	s.recordLoginFailureWithReason(r, actorID, email, "")
}

// recordLoginFailureWithReason is the structured variant of recordLoginFailure.
// Kept as a sibling (rather than changing the existing call sites' signature)
// so all four current invocations continue to compile while still emitting
// {email, failure_reason, attempt_count}. attempt_count comes from the
// in-memory LockoutManager — best-effort, drops to 0 if unwired.
func (s *Server) recordLoginFailureWithReason(r *http.Request, actorID, email, failureReason string) {
	if s == nil || s.AuditLogger == nil {
		return
	}
	attemptCount := 0
	if s.LockoutManager != nil {
		attemptCount = s.LockoutManager.FailureCount(email)
	}
	meta, _ := json.Marshal(map[string]any{
		"email":          email,
		"failure_reason": failureReason,
		"attempt_count":  attemptCount,
	})
	_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
		ActorID:    actorID,
		ActorType:  "user",
		Action:     "user.login",
		TargetType: "user",
		TargetID:   actorID,
		IP:         r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		Metadata:   string(meta),
		Status:     "failure",
	})
}

// writeJSON encodes data as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //#nosec G104 -- write to ResponseWriter; no actionable recovery //nolint:errcheck
}

// userResponseMap converts a userResponse to a map[string]interface{} so JWT
// fields can be merged before writing the final response.
func userResponseMap(u userResponse) map[string]interface{} {
	return map[string]interface{}{
		"id":            u.ID,
		"email":         u.Email,
		"emailVerified": u.EmailVerified,
		"name":          u.Name,
		"avatarUrl":     u.AvatarURL,
		"mfaEnabled":    u.MFAEnabled,
		"createdAt":     u.CreatedAt,
		"updatedAt":     u.UpdatedAt,
	}
}
