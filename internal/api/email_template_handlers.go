// Package api — admin-key-authenticated email template CRUD, preview, send-test,
// and reset handlers. Part of Phase A (task A7) of the branding + hosted-
// components plan. All routes live under /admin/email-templates with the same
// AdminAPIKeyFromStore middleware gate as the rest of the admin group.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// isEmailTemplateNotFound detects the sentinel error the storage layer wraps
// around sql.ErrNoRows for template lookups. The storage package returns a
// fmt.Errorf("email template not found: %s", id) that doesn't carry an Is()
// implementation, so a prefix match is the least-bad bridge until the storage
// contract exposes a typed error.
func isEmailTemplateNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "email template not found")
}

// defaultSampleData is the fallback variable bag used when a preview or
// send-test caller doesn't supply `sample_data`. Templates reference
// AppName, Link, and UserEmail via {{.Field}} syntax; keys not present in
// a given template are silently ignored by the Go template engine, so this
// single struct safely feeds all 5 V1 templates.
func defaultSampleData() map[string]any {
	return map[string]any{
		"AppName":   "SharkAuth",
		"Link":      "https://example.com/verify/abc",
		"UserEmail": "user@example.com",
	}
}

// handleListEmailTemplates handles GET /admin/email-templates.
// Returns all seeded templates in a {data: [...]} envelope.
func (s *Server) handleListEmailTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListEmailTemplates(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	if list == nil {
		list = []*storage.EmailTemplate{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

// handleGetEmailTemplate handles GET /admin/email-templates/{id}. 404s on miss.
func (s *Server) handleGetEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tmpl, err := s.Store.GetEmailTemplate(r.Context(), id)
	if err != nil {
		if isEmailTemplateNotFound(err) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Email template not found"))
			return
		}
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

// handlePatchEmailTemplate handles PATCH /admin/email-templates/{id}.
// Storage applies its own field allowlist; unknown keys are silently dropped.
// Returns the freshly-read template via handleGetEmailTemplate so callers
// never have to re-GET after a save.
func (s *Server) handlePatchEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Short-circuit if the row doesn't exist, so unknown ids return a real
	// 404 rather than a 200 with an empty update.
	if _, err := s.Store.GetEmailTemplate(r.Context(), id); err != nil {
		if isEmailTemplateNotFound(err) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Email template not found"))
			return
		}
		internal(w, err)
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if err := s.Store.UpdateEmailTemplate(r.Context(), id, body); err != nil {
		internal(w, err)
		return
	}
	s.handleGetEmailTemplate(w, r)
}

// previewRequest is the shared shape for /preview and /send-test input bodies.
// `config` optionally overrides the resolved branding; `sample_data` optionally
// overrides the default variable bag.
type previewRequest struct {
	Config     *storage.BrandingConfig `json:"config,omitempty"`
	SampleData map[string]any          `json:"sample_data,omitempty"`
	ToEmail    string                  `json:"to_email,omitempty"`
}

// handlePreviewEmailTemplate handles POST /admin/email-templates/{id}/preview.
// Renders the template with optional branding override + sample data defaults
// so the dashboard can drop the HTML into a sandboxed iframe.
func (s *Server) handlePreviewEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tmpl, err := s.Store.GetEmailTemplate(r.Context(), id)
	if err != nil {
		if isEmailTemplateNotFound(err) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Email template not found"))
			return
		}
		internal(w, err)
		return
	}

	// Empty body is valid — decode best-effort so PATCH-style empty requests
	// still hit the default-branding + default-sample-data path.
	var req previewRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
			return
		}
	}

	branding := req.Config
	if branding == nil {
		resolved, err := s.Store.ResolveBranding(r.Context(), "")
		if err != nil {
			internal(w, err)
			return
		}
		branding = resolved
	}

	data := req.SampleData
	if len(data) == 0 {
		data = defaultSampleData()
	}

	rendered, err := email.RenderStructured(tmpl, branding, data)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"html":    rendered.HTML,
		"subject": rendered.Subject,
	})
}

// handleSendTestEmail handles POST /admin/email-templates/{id}/send-test.
// Renders the template with default sample data + resolved branding and hands
// the result to the MagicLinkManager's underlying sender — the same pipe the
// rest of the auth flows use, so a working /send-test proves the whole email
// stack end to end.
//
// Per spec OQ4 the `to_email` field is the only validation; no regex / DNS /
// allowlist check is applied. Admins can target internal fixtures or their own
// mailbox at will.
func (s *Server) handleSendTestEmail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tmpl, err := s.Store.GetEmailTemplate(r.Context(), id)
	if err != nil {
		if isEmailTemplateNotFound(err) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Email template not found"))
			return
		}
		internal(w, err)
		return
	}

	var req previewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.ToEmail == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Field 'to_email' is required"))
		return
	}

	branding, err := s.Store.ResolveBranding(r.Context(), "")
	if err != nil {
		internal(w, err)
		return
	}
	data := defaultSampleData()

	rendered, err := email.RenderStructured(tmpl, branding, data)
	if err != nil {
		internal(w, err)
		return
	}

	var sender email.Sender
	if s.MagicLinkManager != nil {
		sender = s.MagicLinkManager.Sender()
	}
	if sender == nil {
		writeJSON(w, http.StatusServiceUnavailable, errPayload("send_failed", "No email provider configured"))
		return
	}

	if err := sender.Send(&email.Message{
		To:      req.ToEmail,
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
	}); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("send_failed", "Failed to send test email: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "sent"})
}

// handleResetEmailTemplate handles POST /admin/email-templates/{id}/reset.
// Looks up the V1 seed by id and UPDATEs the row with seed values — we
// intentionally do NOT re-INSERT-OR-IGNORE because that's a no-op once the
// row exists, and the point of reset is to overwrite user edits.
func (s *Server) handleResetEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var seed *storage.EmailTemplate
	for _, candidate := range storage.DefaultEmailTemplateSeedsForExport() {
		if candidate.ID == id {
			seed = candidate
			break
		}
	}
	if seed == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Email template not found"))
		return
	}

	// Confirm the row actually exists before attempting an UPDATE — otherwise
	// the UPDATE silently matches zero rows and the caller can't tell.
	if _, err := s.Store.GetEmailTemplate(r.Context(), id); err != nil {
		if isEmailTemplateNotFound(err) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Email template not found"))
			return
		}
		internal(w, err)
		return
	}

	fields := map[string]any{
		"subject":          seed.Subject,
		"preheader":        seed.Preheader,
		"header_text":      seed.HeaderText,
		"body_paragraphs":  seed.BodyParagraphs,
		"cta_text":         seed.CTAText,
		"cta_url_template": seed.CTAURLTemplate,
		"footer_text":      seed.FooterText,
	}
	if err := s.Store.UpdateEmailTemplate(r.Context(), id, fields); err != nil {
		internal(w, err)
		return
	}
	s.handleGetEmailTemplate(w, r)
}
