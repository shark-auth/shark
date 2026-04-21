// Package api — structured error envelope (W18).
//
// ErrorResponse is the canonical JSON error shape for all /auth/** and
// /admin/** endpoints. Every field except Error and Message is optional so
// handlers can opt in to richer metadata without breaking existing clients
// that only read {error, message}.
//
// OAuth endpoints under /oauth/** MUST NOT use this type — they emit the
// RFC 6749 §5.2 shape defined in internal/oauth/errors.go so client libraries
// (e.g. AppAuth, oauth2-proxy, standard SDKs) keep working unmodified.
package api

import (
	"encoding/json"
	"net/http"
)

// docsURLBase is the public-docs prefix used when WithDocsURL(code) is called.
// The actual site is a TODO; the URL is stable so integrators can hard-code
// switch/case on `code` today and get a live link when the site ships.
const docsURLBase = "https://docs.shark-auth.com/errors/"

// ErrorResponse is the structured error envelope for non-OAuth endpoints.
//
//	{
//	  "error":   "invalid_request",     // short kebab/snake slug (legacy, kept for BC)
//	  "message": "Human readable",      // end-user safe message
//	  "code":    "password_too_short",  // machine-readable discriminator
//	  "docs_url":"https://docs.shark-auth.com/errors/password_too_short",
//	  "details":{"min_length":12}       // optional structured context
//	}
//
// `error` and `code` are often the same for simple cases; they diverge when
// one top-level class (e.g. `weak_password`) fans out into multiple codes
// (`password_too_short`, `password_too_common`, `password_in_breach`).
type ErrorResponse struct {
	Error   string         `json:"error"`
	Message string         `json:"message"`
	Code    string         `json:"code"`
	DocsURL string         `json:"docs_url,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// NewError builds an ErrorResponse. `code` populates both `error` and `code`
// by default; use the explicit struct literal if you need them to differ
// (e.g. legacy `weak_password` class with a refined code).
func NewError(code, message string) ErrorResponse {
	return ErrorResponse{
		Error:   code,
		Message: message,
		Code:    code,
	}
}

// WithDocsURL sets DocsURL to the canonical per-code documentation link.
// Passing the same `code` as NewError is the common case; an explicit
// override is supported so a handler can point at a class-level doc page.
func (e ErrorResponse) WithDocsURL(code string) ErrorResponse {
	e.DocsURL = docsURLBase + code
	return e
}

// WithDetails attaches structured context (e.g. min_length, retry_after).
// The returned value replaces Details wholesale; nil clears it.
func (e ErrorResponse) WithDetails(kv map[string]any) ErrorResponse {
	e.Details = kv
	return e
}

// WriteError serialises the envelope as JSON with the given status code.
// Content-Type is forced to application/json. Encoding errors are ignored —
// the response has already been committed and there is no actionable recovery.
func WriteError(w http.ResponseWriter, status int, err ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(err) //#nosec G104 -- response already committed
}

// Common error codes. Handlers SHOULD reuse these constants rather than
// spelling the strings inline so grep and the docs catalog stay in sync.
const (
	// Generic / HTTP-layer.
	CodeInvalidRequest = "invalid_request"
	CodeUnauthorized   = "unauthorized"
	CodeForbidden      = "forbidden"
	CodeNotFound       = "not_found"
	CodeConflict       = "conflict"
	CodeRateLimited    = "rate_limited"
	CodeInternal       = "internal_error"

	// Auth / credentials.
	CodeInvalidCredentials = "invalid_credentials"
	CodeAccountLocked      = "account_locked"
	CodeMFARequired        = "mfa_required"
	CodeSessionExpired     = "session_expired"
	CodeTokenUsed          = "token_used"
	CodeInvalidToken       = "invalid_token"

	// Password policy — fine-grained so integrators can render targeted UI.
	CodeWeakPassword      = "weak_password"
	CodePasswordTooShort  = "password_too_short"
	CodePasswordTooCommon = "password_too_common"
	CodePasswordInBreach  = "password_in_breach"

	// MFA / passkey flows.
	CodeEnrollmentAlreadyComplete = "enrollment_already_complete"
	CodeChallengeExpired          = "challenge_expired"
	CodeChallengeInvalid          = "challenge_invalid"
	CodeNoMatchingCredential      = "no_matching_credential"

	// Email verification / magic link.
	CodeEmailAlreadyVerified = "email_already_verified"
	CodeMagicLinkExpired     = "magic_link_expired"

	// Admin-flavoured.
	CodeBootstrapLocked = "bootstrap_locked"
	CodeInvalidScope    = "invalid_scope"
)
