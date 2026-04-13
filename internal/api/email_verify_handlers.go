package api

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
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
		log.Printf("ERROR: failed to send verification email to %s: %v", user.Email, err)
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

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}
