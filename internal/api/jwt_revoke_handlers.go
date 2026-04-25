package api

import (
	"encoding/json"
	"net/http"
	"time"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// revokeRequest is the request body for POST /api/v1/auth/revoke.
type revokeRequest struct {
	// Token is optional: if not provided, the current JWT from context is revoked.
	Token string `json:"token,omitempty"`
}

// adminRevokeJTIRequest is the request body for POST /api/v1/admin/auth/revoke-jti.
type adminRevokeJTIRequest struct {
	JTI       string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
}

// handleUserRevoke handles POST /api/v1/auth/revoke.
// Session-gated (cookie or JWT). Revokes the JTI of the current JWT in context,
// or a token passed explicitly in the body. Cookie sessions are unaffected unless
// the caller also holds a JWT.
//
// Per §1.7: if auth method is "jwt", revoke the claims.ID from context.
// If body provides a token, validate it first then revoke its JTI.
func (s *Server) handleUserRevoke(w http.ResponseWriter, r *http.Request) {
	if s.JWTManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "feature_disabled",
			"feature": "jwt_revoke",
			"message": "JWT is not configured. Set jwt.signing_key_path in config and restart.",
			"config":  "jwt.signing_key_path",
		})
		return
	}

	var req revokeRequest
	// Ignore decode errors — body is optional
	_ = json.NewDecoder(r.Body).Decode(&req)

	ctx := r.Context()

	if req.Token != "" {
		// Caller supplied an explicit token to revoke — validate it then revoke its JTI.
		claims, err := s.JWTManager.Validate(ctx, req.Token)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "invalid_token",
				"message": "Token is invalid or expired",
			})
			return
		}
		// Only allow revoking tokens that belong to the authenticated user.
		userID := mw.GetUserID(ctx)
		if claims.Subject != userID {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":   "forbidden",
				"message": "Cannot revoke a token belonging to another user",
			})
			return
		}
		if err := s.JWTManager.RevokeJTI(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal_error",
				"message": "Failed to revoke token",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Token revoked"})
		return
	}

	// No explicit token: revoke the JWT currently in context (if auth method is "jwt").
	if mw.GetAuthMethod(ctx) == "jwt" {
		claims := mw.GetClaims(ctx)
		if claims != nil {
			if err := s.JWTManager.RevokeJTI(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error":   "internal_error",
					"message": "Failed to revoke token",
				})
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Token revoked"})
}

// handleAdminRevokeJTI handles POST /api/v1/admin/auth/revoke-jti.
// Admin-key-gated. Revokes an arbitrary JTI for any user.
func (s *Server) handleAdminRevokeJTI(w http.ResponseWriter, r *http.Request) {
	if s.JWTManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "feature_disabled",
			"feature": "jwt_revoke",
			"message": "JWT is not configured. Set jwt.signing_key_path in config and restart.",
			"config":  "jwt.signing_key_path",
		})
		return
	}

	var req adminRevokeJTIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	if req.JTI == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "jti is required",
		})
		return
	}

	if req.ExpiresAt.IsZero() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "expires_at is required",
		})
		return
	}

	if err := s.JWTManager.RevokeJTI(r.Context(), req.JTI, req.ExpiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to revoke JTI",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "JTI revoked"})
}
