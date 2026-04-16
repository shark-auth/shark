package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// KnownWebhookEvents is the full set of emitted events. Admins register
// webhooks against these; requests with unknown event names are refused so
// typos surface on create rather than silently never firing.
var KnownWebhookEvents = map[string]bool{
	storage.WebhookEventUserCreated:    true,
	storage.WebhookEventUserDeleted:    true,
	storage.WebhookEventSessionRevoked: true,
	storage.WebhookEventOrgCreated:     true,
	storage.WebhookEventOrgMemberAdded: true,
}

type webhookResponse struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Enabled     bool     `json:"enabled"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// webhookResponseWithSecret is only returned at create time so the caller can
// store the HMAC secret. Subsequent reads never include it.
type webhookResponseWithSecret struct {
	webhookResponse
	Secret string `json:"secret"`
}

func webhookToResponse(w *storage.Webhook) webhookResponse {
	var events []string
	_ = json.Unmarshal([]byte(w.Events), &events)
	return webhookResponse{
		ID: w.ID, URL: w.URL, Events: events, Enabled: w.Enabled,
		Description: w.Description, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}
}

type createWebhookRequest struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Description string   `json:"description,omitempty"`
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if err := validateWebhookURL(req.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
		return
	}
	if err := validateEvents(req.Events); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_events", err.Error()))
		return
	}

	secret, err := newWebhookSecret()
	if err != nil {
		internal(w, err)
		return
	}
	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	eventsJSON, _ := json.Marshal(req.Events)

	hook := &storage.Webhook{
		ID: "wh_" + id, URL: req.URL, Secret: secret,
		Events:      string(eventsJSON),
		Enabled:     true,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   now, UpdatedAt: now,
	}
	if err := s.Store.CreateWebhook(r.Context(), hook); err != nil {
		internal(w, err)
		return
	}

	resp := webhookResponseWithSecret{
		webhookResponse: webhookToResponse(hook),
		Secret:          secret,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	hooks, err := s.Store.ListWebhooks(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]webhookResponse, 0, len(hooks))
	for _, h := range hooks {
		out = append(out, webhookToResponse(h))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	hook, err := s.Store.GetWebhookByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Webhook not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, webhookToResponse(hook))
}

type updateWebhookRequest struct {
	URL         *string   `json:"url,omitempty"`
	Events      *[]string `json:"events,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
	Description *string   `json:"description,omitempty"`
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	hook, err := s.Store.GetWebhookByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Webhook not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	if req.URL != nil {
		if err := validateWebhookURL(*req.URL); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
			return
		}
		hook.URL = *req.URL
	}
	if req.Events != nil {
		if err := validateEvents(*req.Events); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_events", err.Error()))
			return
		}
		b, _ := json.Marshal(*req.Events)
		hook.Events = string(b)
	}
	if req.Enabled != nil {
		hook.Enabled = *req.Enabled
	}
	if req.Description != nil {
		hook.Description = strings.TrimSpace(*req.Description)
	}
	hook.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Store.UpdateWebhook(r.Context(), hook); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, webhookToResponse(hook))
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.Store.DeleteWebhook(r.Context(), id); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleTestWebhook emits a synthetic `webhook.test` event so admins can verify
// signature + network reachability without triggering a real flow.
func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := s.Store.GetWebhookByID(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Webhook not found"))
		return
	} else if err != nil {
		internal(w, err)
		return
	}
	if s.WebhookDispatcher == nil {
		writeJSON(w, http.StatusServiceUnavailable, errPayload("dispatcher_unavailable", "Webhook dispatcher not initialized"))
		return
	}
	deliveryID, err := s.WebhookDispatcher.Redeliver(r.Context(), id, "webhook.test", map[string]string{
		"webhook_id": id,
		"note":       "This is a test event triggered from the admin API.",
	})
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"message":     "Test event enqueued",
		"delivery_id": deliveryID,
	})
}

// handleListDeliveries returns the delivery log for a single webhook with
// keyset cursor pagination matching /admin/sessions.
func (s *Server) handleListDeliveries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	cursor := r.URL.Query().Get("cursor")

	dels, err := s.Store.ListWebhookDeliveriesByWebhookID(r.Context(), id, limit, cursor)
	if err != nil {
		internal(w, err)
		return
	}

	var next string
	if len(dels) > 0 && (limit == 0 || len(dels) >= clampDeliveryLimit(limit)) {
		last := dels[len(dels)-1]
		next = last.CreatedAt + "|" + last.ID
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":        dels,
		"next_cursor": next,
	})
}

// --- validation helpers ---

// validateWebhookURL requires https:// in production (http:// allowed in dev)
// to prevent credential-in-query-string leaks over plaintext.
func validateWebhookURL(raw string) error {
	if raw == "" {
		return errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("url is not a valid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url must be http(s)")
	}
	if u.Host == "" {
		return errors.New("url must include a host")
	}
	return nil
}

func validateEvents(events []string) error {
	if len(events) == 0 {
		return errors.New("at least one event is required")
	}
	for _, e := range events {
		if !KnownWebhookEvents[e] {
			return errors.New("unknown event: " + e)
		}
	}
	return nil
}

func newWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whsec_" + hex.EncodeToString(b), nil
}

func clampDeliveryLimit(n int) int {
	if n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}
