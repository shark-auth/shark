// Package oauth — RFC 6749 §5.2 error envelope (W18).
//
// All /oauth/** endpoints MUST emit this shape. Client libraries in the wild
// (AppAuth-iOS/Android, oauth2-proxy, Go's x/oauth2, Authlib, etc.) assume the
// standard `{error, error_description?, error_uri?}` body — any deviation
// breaks them. The Shark-specific envelope in internal/api/errors.go does NOT
// apply here.
package oauth

import (
	"encoding/json"
	"net/http"
)

// OAuthError is the RFC 6749 §5.2 error response body. `error` is a single
// code from the RFC registry (see constants below); the other two fields are
// optional free-form strings. No additional top-level fields may be added —
// any Shark extensions MUST piggy-back on error_uri or a separate header.
type OAuthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// WriteOAuthError writes the RFC 6749 §5.2 response body with the mandated
// cache headers. Callers choose the HTTP status — typically 400 for bad
// requests, 401 for invalid_client/invalid_token, 403 for insufficient_scope.
func WriteOAuthError(w http.ResponseWriter, status int, err OAuthError) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(err) //#nosec G104 -- response already committed
}

// NewOAuthError is a convenience builder.
func NewOAuthError(code, description string) OAuthError {
	return OAuthError{Error: code, ErrorDescription: description}
}

// RFC 6749 §5.2 + extensions (OIDC, RFC 7009 revoke, RFC 7662 introspect,
// RFC 8628 device, RFC 9449 DPoP). Reuse these instead of string literals.
const (
	ErrInvalidRequest        = "invalid_request"
	ErrInvalidClient         = "invalid_client"
	ErrInvalidGrant          = "invalid_grant"
	ErrUnauthorizedClient    = "unauthorized_client"
	ErrUnsupportedGrantType  = "unsupported_grant_type"
	ErrInvalidScope          = "invalid_scope"
	ErrAccessDenied          = "access_denied"
	ErrServerError           = "server_error"
	ErrTemporarilyUnavail    = "temporarily_unavailable"
	ErrInvalidToken          = "invalid_token"
	ErrUnsupportedTokenType  = "unsupported_token_type"
	ErrInvalidDPoPProof      = "invalid_dpop_proof" // RFC 9449
	ErrInteractionRequired   = "interaction_required"
	ErrLoginRequired         = "login_required"
	ErrConsentRequired       = "consent_required"
	ErrAuthorizationPending  = "authorization_pending" // RFC 8628
	ErrSlowDown              = "slow_down"             // RFC 8628
	ErrExpiredToken          = "expired_token"
)
