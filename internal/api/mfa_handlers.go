package api

import (
	"encoding/json"
	"net/http"
	"time"

	mw "github.com/shark-auth/shark/internal/api/middleware"
	"github.com/shark-auth/shark/internal/auth"
)

// mfaEnrollResponse is the response body for POST /api/v1/auth/mfa/enroll.
type mfaEnrollResponse struct {
	Secret string `json:"secret"`
	QRURI  string `json:"qr_uri"`
}

// mfaVerifyRequest is the request body for POST /api/v1/auth/mfa/verify.
type mfaVerifyRequest struct {
	Code string `json:"code"`
}

// mfaChallengeRequest is the request body for POST /api/v1/auth/mfa/challenge.
type mfaChallengeRequest struct {
	Code string `json:"code"`
}

// mfaRecoveryRequest is the request body for POST /api/v1/auth/mfa/recovery.
type mfaRecoveryRequest struct {
	Code string `json:"code"`
}

// mfaDisableRequest is the request body for DELETE /api/v1/auth/mfa.
type mfaDisableRequest struct {
	Code string `json:"code"`
}

// handleMFAEnroll generates a new TOTP secret for the authenticated user.
// POST /api/v1/auth/mfa/enroll
// Requires: authenticated session with mfa_passed=true (fully authenticated).
func (s *Server) handleMFAEnroll(w http.ResponseWriter, r *http.Request) {
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

	// Block re-enroll only when already verified (MFAVerifiedAt is set).
	// Allow re-enroll when the user has a pending secret (enrolled but not yet verified).
	if user.MFAVerifiedAt != nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":   "mfa_already_enabled",
			"message": "MFA is already enabled for this account",
		})
		return
	}

	mfaMgr := auth.NewMFAManager(s.Store, s.Config.MFA)
	secret, qrURI, err := mfaMgr.GenerateSecret(user.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to generate TOTP secret",
		})
		return
	}

	// Encrypt and store the secret on the user (not yet verified)
	encSecret, err := s.FieldEncryptor.Encrypt(secret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}
	user.MFASecret = &encSecret
	user.MFAVerified = false
	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}
	s.evictSessionAuth(mw.GetSessionID(r.Context()))

	writeJSON(w, http.StatusOK, mfaEnrollResponse{
		Secret: secret,
		QRURI:  qrURI,
	})
}

// handleMFAVerify confirms MFA setup by validating the first TOTP code.
// POST /api/v1/auth/mfa/verify
// Requires: authenticated session, user must have a pending MFA secret (enrolled but not verified).
func (s *Server) handleMFAVerify(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "No valid session",
		})
		return
	}

	var req mfaVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid or missing TOTP code",
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

	if user.MFASecret == nil || *user.MFASecret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "mfa_not_enrolled",
			"message": "Must enroll in MFA first",
		})
		return
	}

	if user.MFAVerifiedAt != nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error":   "mfa_already_verified",
			"message": "MFA is already verified and enabled",
		})
		return
	}

	// Decrypt MFA secret for validation
	decSecret, err := s.FieldEncryptor.Decrypt(*user.MFASecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	mfaMgr := auth.NewMFAManager(s.Store, s.Config.MFA)
	if !mfaMgr.ValidateTOTP(decSecret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_code",
			"message": "Invalid TOTP code",
		})
		return
	}

	// Enable MFA on the user and stamp the verified_at time (F3.2).
	now := time.Now().UTC().Format(time.RFC3339)
	user.MFAEnabled = true
	user.MFAVerified = true
	user.MFAVerifiedAt = &now
	user.UpdatedAt = now
	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Generate recovery codes
	codes, err := mfaMgr.GenerateRecoveryCodes(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to generate recovery codes",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mfa_enabled":    true,
		"recovery_codes": codes,
	})
}

// handleMFAChallenge verifies a TOTP code during login (partial session).
// POST /api/v1/auth/mfa/challenge
// Requires: session with mfa_passed=false (partial session from login with MFA).
func (s *Server) handleMFAChallenge(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	sessionID := mw.GetSessionID(r.Context())
	if userID == "" || sessionID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "No valid session",
		})
		return
	}

	// MFA challenge is only for partial sessions (mfa_passed=false)
	if mw.GetMFAPassed(r.Context()) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "mfa_already_passed",
			"message": "MFA has already been completed for this session",
		})
		return
	}

	var req mfaChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid or missing TOTP code",
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

	if !user.MFAEnabled || user.MFASecret == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "mfa_not_enabled",
			"message": "MFA is not enabled for this account",
		})
		return
	}

	// Decrypt MFA secret for validation
	decSecret, err := s.FieldEncryptor.Decrypt(*user.MFASecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	mfaMgr := auth.NewMFAManager(s.Store, s.Config.MFA)
	if !mfaMgr.ValidateTOTP(decSecret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_code",
			"message": "Invalid TOTP code",
		})
		return
	}

	// Upgrade session to mfa_passed=true
	if err := s.SessionManager.UpgradeMFA(r.Context(), sessionID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to upgrade session",
		})
		return
	}
	s.evictSessionAuth(sessionID)

	writeJSON(w, http.StatusOK, userToResponse(user))
}

// handleMFARecovery uses a recovery code during login (partial session).
// POST /api/v1/auth/mfa/recovery
// Requires: session with mfa_passed=false (partial session from login with MFA).
func (s *Server) handleMFARecovery(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	sessionID := mw.GetSessionID(r.Context())
	if userID == "" || sessionID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "No valid session",
		})
		return
	}

	// Recovery is only for partial sessions (mfa_passed=false)
	if mw.GetMFAPassed(r.Context()) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "mfa_already_passed",
			"message": "MFA has already been completed for this session",
		})
		return
	}

	var req mfaRecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid or missing recovery code",
		})
		return
	}

	mfaMgr := auth.NewMFAManager(s.Store, s.Config.MFA)
	ok, err := mfaMgr.VerifyRecoveryCode(r.Context(), userID, req.Code)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_code",
			"message": "Invalid recovery code",
		})
		return
	}

	// Upgrade session to mfa_passed=true
	if err := s.SessionManager.UpgradeMFA(r.Context(), sessionID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to upgrade session",
		})
		return
	}
	s.evictSessionAuth(sessionID)

	user, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

// handleMFADisable disables MFA for the authenticated user.
// DELETE /api/v1/auth/mfa
// Requires: authenticated session with mfa_passed=true, current TOTP code in body.
func (s *Server) handleMFADisable(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "No valid session",
		})
		return
	}

	var req mfaDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid or missing TOTP code",
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

	if !user.MFAEnabled || user.MFASecret == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "mfa_not_enabled",
			"message": "MFA is not enabled for this account",
		})
		return
	}

	// Decrypt MFA secret for validation
	decSecret, err := s.FieldEncryptor.Decrypt(*user.MFASecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	mfaMgr := auth.NewMFAManager(s.Store, s.Config.MFA)
	if !mfaMgr.ValidateTOTP(decSecret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_code",
			"message": "Invalid TOTP code",
		})
		return
	}

	// Disable MFA â€” clear verified_at so a future enroll is treated as fresh.
	user.MFAEnabled = false
	user.MFAVerified = false
	user.MFASecret = nil
	user.MFAVerifiedAt = nil
	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Internal server error",
		})
		return
	}

	// Delete recovery codes
	_ = s.Store.DeleteAllMFARecoveryCodesByUserID(r.Context(), userID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mfa_enabled": false,
	})
}

// handleMFARecoveryCodes regenerates recovery codes for the authenticated user.
// GET /api/v1/auth/mfa/recovery-codes
// Requires: authenticated session with mfa_passed=true, MFA must be enabled.
func (s *Server) handleMFARecoveryCodes(w http.ResponseWriter, r *http.Request) {
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

	if !user.MFAEnabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "mfa_not_enabled",
			"message": "MFA is not enabled for this account",
		})
		return
	}

	mfaMgr := auth.NewMFAManager(s.Store, s.Config.MFA)
	codes, err := mfaMgr.GenerateRecoveryCodes(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to generate recovery codes",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"recovery_codes": codes,
	})
}
