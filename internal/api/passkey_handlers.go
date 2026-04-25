package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/auth"
	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// passkeyLoginBeginRequest is the request body for POST /passkey/login/begin.
type passkeyLoginBeginRequest struct {
	Email string `json:"email"`
}

// passkeyRenameRequest is the request body for PATCH /passkey/credentials/{id}.
type passkeyRenameRequest struct {
	Name string `json:"name"`
}

// passkeyDisabled writes a 503 when the passkey manager is not configured
// (NewPasskeyManager failed at startup, e.g. missing webauthn.rp_id). Caller
// must return when this returns true. Avoids the prior silent nil-deref 500
// panics on the begin/finish endpoints.
func (s *Server) passkeyDisabled(w http.ResponseWriter) bool {
	if s.PasskeyManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "feature_disabled",
			"feature": "passkey",
			"message": "Passkeys are not configured. Set webauthn.rp_id and webauthn.rp_origin in config and restart.",
			"config":  "webauthn.rp_id, webauthn.rp_origin",
		})
		return true
	}
	return false
}

// passkeyCredentialResponse is the JSON response for a passkey credential.
type passkeyCredentialResponse struct {
	ID         string  `json:"id"`
	Name       *string `json:"name,omitempty"`
	Transports string  `json:"transports"`
	BackedUp   bool    `json:"backed_up"`
	CreatedAt  string  `json:"created_at"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
}

func (s *Server) handlePasskeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if s.passkeyDisabled(w) {
		return
	}
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "Authentication required",
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

	creation, challengeKey, err := s.PasskeyManager.BeginRegistration(r.Context(), user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to start passkey registration",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"publicKey":    creation.Response,
		"challengeKey": challengeKey,
	})
}

func (s *Server) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if s.passkeyDisabled(w) {
		return
	}
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	challengeKey := r.Header.Get("X-Challenge-Key")
	if challengeKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing X-Challenge-Key header",
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

	pkCred, err := s.PasskeyManager.FinishRegistration(r.Context(), user, challengeKey, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "registration_failed",
			"message": "Failed to verify passkey registration",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"credential_id": pkCred.ID,
		"name":          pkCred.Name,
	})
}

func (s *Server) handlePasskeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	if s.passkeyDisabled(w) {
		return
	}
	var req passkeyLoginBeginRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "invalid_request",
				"message": "Invalid JSON body",
			})
			return
		}
	}

	assertion, challengeKey, err := s.PasskeyManager.BeginLogin(r.Context(), req.Email)
	if err != nil {
		if err == auth.ErrNoPasskeys {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "no_passkeys",
				"message": "User has no registered passkeys",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to start passkey login",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"publicKey":    assertion.Response,
		"challengeKey": challengeKey,
	})
}

func (s *Server) handlePasskeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	if s.passkeyDisabled(w) {
		return
	}
	challengeKey := r.Header.Get("X-Challenge-Key")
	if challengeKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing X-Challenge-Key header",
		})
		return
	}

	user, sess, err := s.PasskeyManager.FinishLogin(r.Context(), challengeKey, r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "authentication_failed",
			"message": "Failed to verify passkey",
		})
		return
	}

	s.SessionManager.SetSessionCookie(w, sess.ID)

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (s *Server) handlePasskeyCredentialsList(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	creds, err := s.Store.GetPasskeysByUserID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to list passkey credentials",
		})
		return
	}

	result := make([]passkeyCredentialResponse, len(creds))
	for i, c := range creds {
		result[i] = passkeyCredentialResponse{
			ID:         c.ID,
			Name:       c.Name,
			Transports: c.Transports,
			BackedUp:   c.BackedUp,
			CreatedAt:  c.CreatedAt,
			LastUsedAt: c.LastUsedAt,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"credentials": result,
	})
}

func (s *Server) handlePasskeyCredentialDelete(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	credID := chi.URLParam(r, "id")
	if credID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing credential ID",
		})
		return
	}

	// Verify the credential belongs to the user by listing their creds
	creds, err := s.Store.GetPasskeysByUserID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to verify credential ownership",
		})
		return
	}

	found := false
	for _, c := range creds {
		if c.ID == credID {
			found = true
			break
		}
	}

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Passkey credential not found",
		})
		return
	}

	if err := s.Store.DeletePasskeyCredential(r.Context(), credID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to delete passkey credential",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (s *Server) handlePasskeyCredentialRename(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	credID := chi.URLParam(r, "id")
	if credID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Missing credential ID",
		})
		return
	}

	var req passkeyRenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Name is required",
		})
		return
	}

	// Verify the credential belongs to the user
	creds, err := s.Store.GetPasskeysByUserID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to verify credential ownership",
		})
		return
	}

	var target *passkeyCredentialResponse
	for _, c := range creds {
		if c.ID == credID {
			c.Name = &req.Name
			if err := s.Store.UpdatePasskeyCredential(r.Context(), c); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error":   "internal_error",
					"message": "Failed to rename passkey credential",
				})
				return
			}
			target = &passkeyCredentialResponse{
				ID:         c.ID,
				Name:       c.Name,
				Transports: c.Transports,
				BackedUp:   c.BackedUp,
				CreatedAt:  c.CreatedAt,
				LastUsedAt: c.LastUsedAt,
			}
			break
		}
	}

	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Passkey credential not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, target)
}
