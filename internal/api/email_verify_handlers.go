package api

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// handleEmailVerifySend handles POST /api/v1/auth/email/verify/send
// Sends a verification email to the authenticated user.
func (s *Server) handleEmailVerifySend(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	if user.EmailVerified {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Email is already verified",
		})
		return
	}

	if s.MagicLinkManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "email_not_configured",
			"message": "Email sending is not configured",
		})
		return
	}

	if err := s.MagicLinkManager.SendEmailVerification(r.Context(), user.Email); err != nil {
		slog.Error("failed to send verification email", "email", user.Email, "error", err)
	}

	// Always return success to avoid timing attacks
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent",
	})
}

// handleEmailVerify handles GET /api/v1/auth/email/verify?token=...
// Verifies the email using the token from the verification email.
func (s *Server) handleEmailVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing token parameter",
		})
		return
	}

	if s.MagicLinkManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "email_not_configured",
			"message": "Email sending is not configured",
		})
		return
	}

	emailAddr, err := s.MagicLinkManager.VerifyEmailToken(r.Context(), token)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_token",
			"message": "Invalid or expired verification token",
		})
		return
	}

	user, err := s.Store.GetUserByEmail(r.Context(), emailAddr)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_token",
			"message": "Invalid or expired verification token",
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

	if !user.EmailVerified {
		user.EmailVerified = true
		user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := s.Store.UpdateUser(r.Context(), user); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal_error",
				"message": "Internal server error",
			})
			return
		}
	}

	// Fire welcome email once — idempotent via UPDATE ... WHERE welcome_email_sent = 0
	// at the store layer. If MarkWelcomeEmailSent returns sql.ErrNoRows the
	// flag was already set by a prior verification (or the user row is gone),
	// so we skip the send entirely. Any DB errors other than ErrNoRows are
	// also silently skipped — a missed welcome isn't worth failing the
	// verify response on.
	if err := s.Store.MarkWelcomeEmailSent(r.Context(), user.ID); err == nil {
		branding, _ := s.Store.ResolveBranding(r.Context(), "")
		dashboardURL := strings.TrimRight(s.Config.Server.BaseURL, "/") + "/admin"
		rendered, rErr := email.RenderWelcome(r.Context(), s.Store, branding, email.WelcomeData{
			AppName:      s.Config.MFA.Issuer,
			UserEmail:    user.Email,
			DashboardURL: dashboardURL,
		})
		if rErr == nil {
			if sender := s.emailSender(); sender != nil {
				to := user.Email
				subject := rendered.Subject
				html := rendered.HTML
				userID := user.ID
				go func() {
					// Detached from the request ctx so the goroutine survives
					// past response write. The Sender interface is synchronous
					// and context-free (Send takes only *Message), so no
					// cancellation plumbing is needed here.
					if sendErr := sender.Send(&email.Message{
						To:      to,
						Subject: subject,
						HTML:    html,
					}); sendErr != nil {
						slog.Warn("welcome email send failed", "user_id", userID, "error", sendErr)
					}
				}()
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

// handleAdminEmailVerifySend handles POST /api/v1/users/{id}/verify/send (admin)
// Sends a verification email to a specific user.
func (s *Server) handleAdminEmailVerifySend(w http.ResponseWriter, r *http.Request) {
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

	if user.EmailVerified {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "already_verified",
			"message": "User email is already verified",
		})
		return
	}

	if s.MagicLinkManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "email_not_configured",
			"message": "Email sending is not configured",
		})
		return
	}

	if err := s.MagicLinkManager.SendEmailVerification(r.Context(), user.Email); err != nil {
		slog.Error("admin failed to send verification email", "user_id", id, "email", user.Email, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "send_failed",
			"message": "Failed to send verification email: " + err.Error(),
		})
		return
	}

	if s.AuditLogger != nil {
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			Action:     "admin.user.verification_sent",
			TargetType: "user",
			TargetID:   id,
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent",
	})
}
