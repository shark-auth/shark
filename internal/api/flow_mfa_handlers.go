package api

import (
	"encoding/json"
	"net/http"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/authflow"
	"github.com/sharkauth/sharkauth/internal/config"
)

// flowMFAVerifyRequest is the body for POST /api/v1/auth/flow/mfa/verify.
type flowMFAVerifyRequest struct {
	// FlowRunID is informational — not validated today; included so SDK callers
	// can be forward-compatible when we add stateful flow-run resumption.
	FlowRunID   string `json:"flow_run_id"`
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
	UserID      string `json:"user_id"`
}

// handleFlowMFAVerify verifies a TOTP code for a require_mfa_challenge step
// that returned awaiting_mfa.
//
// POST /api/v1/auth/flow/mfa/verify
//
// The endpoint is intentionally unauthenticated (no session required) because
// it is called at the point in the auth flow where the session has not yet
// been fully established. The challenge ID is the proof of prior auth.
//
// Flow:
//  1. Client receives {outcome: "awaiting_mfa", challenge_id: "mfac_..."} from
//     the auth handler that ran the flow.
//  2. User submits TOTP code; client POSTs {challenge_id, code, user_id}.
//  3. This handler:
//     a. Consumes the challenge (expires it atomically).
//     b. Validates the TOTP code.
//     c. Returns 200 {verified: true} on success; 401 on failure.
//
// The HTTP layer in auth_handlers.go is responsible for continuing whatever
// action (session creation, token issue) was deferred during the flow pause.
// For now, returning 200 is the contract: the SDK re-POSTs the original action.
func (s *Server) handleFlowMFAVerify(w http.ResponseWriter, r *http.Request) {
	var req flowMFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.ChallengeID == "" || req.Code == "" || req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "challenge_id, code, and user_id are required"))
		return
	}

	// Consume the challenge — single-use, expires on consumption.
	if !authflow.GlobalChallengeStore.Consume(req.ChallengeID, req.UserID) {
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_challenge", "Challenge not found, expired, or user mismatch"))
		return
	}

	// Look up user to get their MFA secret.
	user, err := s.Store.GetUserByID(r.Context(), req.UserID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_challenge", "User not found"))
		return
	}
	if !user.MFAEnabled || !user.MFAVerified || user.MFASecret == nil {
		writeJSON(w, http.StatusUnauthorized, errPayload("mfa_not_enrolled", "User has no MFA enrolled"))
		return
	}

	decSecret, err := s.FieldEncryptor.Decrypt(*user.MFASecret)
	if err != nil {
		internal(w, err)
		return
	}

	mfaMgr := auth.NewMFAManager(s.Store, config.MFAConfig{})
	if !mfaMgr.ValidateTOTP(decSecret, req.Code) {
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_code", "Invalid or expired TOTP code"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verified": true,
		"user_id":  req.UserID,
	})
}
