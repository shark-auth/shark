// Package api — email redirect URL config handlers.
//
// GET  /admin/email-config  — returns the current redirect URL settings.
// PATCH /admin/email-config — updates one or more redirect URL fields.
//
// These settings tell SharkAuth where to send users after consuming email
// tokens (verify-email, password-reset, magic-link). Devs point these at
// their own hosted pages so they can wire the flows without waiting for
// shark's built-in hosted/* pages to ship.
//
// Storage: merged into the system_config JSON blob under key "email_config".
// Requires AdminAPIKeyFromStore middleware (inherited from /admin group).
package api

import (
	"encoding/json"
	"net/http"
)

// emailConfig holds the redirect URL settings surfaced in the Branding > Email
// > Redirect URLs panel.
type emailConfig struct {
	VerifyRedirectURL    string `json:"verify_redirect_url"`
	ResetRedirectURL     string `json:"reset_redirect_url"`
	MagicLinkRedirectURL string `json:"magic_link_redirect_url"`
}

// sysConfigEmailWrapper is the envelope we merge into the existing system_config
// blob. Other keys are preserved via RawMessage.
type sysConfigEmailWrapper struct {
	EmailConfig *emailConfig `json:"email_config,omitempty"`
}

// handleGetEmailConfig returns the current email redirect URL config.
// GET /admin/email-config
func (s *Server) handleGetEmailConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadEmailConfig(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("internal_error", "Failed to load email config"))
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handlePatchEmailConfig merges supplied fields into the stored config.
// PATCH /admin/email-config
func (s *Server) handlePatchEmailConfig(w http.ResponseWriter, r *http.Request) {
	existing, err := s.loadEmailConfig(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("internal_error", "Failed to load email config"))
		return
	}

	var patch emailConfig
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	// Merge: only overwrite non-empty strings so callers can PATCH a single field.
	if patch.VerifyRedirectURL != "" {
		existing.VerifyRedirectURL = patch.VerifyRedirectURL
	}
	if patch.ResetRedirectURL != "" {
		existing.ResetRedirectURL = patch.ResetRedirectURL
	}
	if patch.MagicLinkRedirectURL != "" {
		existing.MagicLinkRedirectURL = patch.MagicLinkRedirectURL
	}

	// Read the full blob, merge our key, write back.
	rawBlob, _ := s.Store.GetSystemConfig(r.Context())
	merged := make(map[string]json.RawMessage)
	if rawBlob != "" && rawBlob != "{}" {
		_ = json.Unmarshal([]byte(rawBlob), &merged)
	}
	emailBytes, err := json.Marshal(existing)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("internal_error", "Failed to encode email config"))
		return
	}
	merged["email_config"] = json.RawMessage(emailBytes)

	if err := s.Store.SetSystemConfig(r.Context(), merged); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("internal_error", "Failed to save email config"))
		return
	}

	writeJSON(w, http.StatusOK, existing)
}

// loadEmailConfig reads the email_config sub-key from system_config. Returns
// an empty struct on fresh installs (no redirect URLs configured yet).
func (s *Server) loadEmailConfig(r *http.Request) (emailConfig, error) {
	var cfg emailConfig
	raw, err := s.Store.GetSystemConfig(r.Context())
	if err != nil || raw == "" || raw == "{}" {
		return cfg, nil
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		return cfg, nil // corrupt blob — return empty rather than 500
	}
	if emailRaw, ok := wrapper["email_config"]; ok {
		_ = json.Unmarshal(emailRaw, &cfg)
	}
	return cfg, nil
}
