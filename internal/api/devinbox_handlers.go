package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// devInboxListResponse is the dashboard-facing shape for listing captured emails.
type devInboxListResponse struct {
	Data []devEmailResponse `json:"data"`
}

type devEmailResponse struct {
	ID        string `json:"id"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	HTML      string `json:"html"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// devInboxAvailable returns true when the runtime email provider is "dev".
// This is the single source of truth for dev-inbox visibility: it follows
// the DB-backed email.provider config (W17), not the legacy --dev flag.
func (s *Server) devInboxAvailable() bool {
	return s.Config != nil && s.Config.Email.Provider == "dev"
}

func (s *Server) handleListDevEmails(w http.ResponseWriter, r *http.Request) {
	if !s.devInboxAvailable() {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Dev inbox is only available when email.provider is set to 'dev'",
		})
		return
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	emails, err := s.Store.ListDevEmails(r.Context(), limit)
	if err != nil {
		internal(w, err)
		return
	}
	resp := devInboxListResponse{Data: make([]devEmailResponse, 0, len(emails))}
	for _, e := range emails {
		resp.Data = append(resp.Data, devEmailResponse{
			ID: e.ID, To: e.To, Subject: e.Subject,
			HTML: e.HTML, Text: e.Text, CreatedAt: e.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetDevEmail(w http.ResponseWriter, r *http.Request) {
	if !s.devInboxAvailable() {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Dev inbox is only available when email.provider is set to 'dev'",
		})
		return
	}
	id := chi.URLParam(r, "id")
	e, err := s.Store.GetDevEmail(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found", "message": "Email not found"})
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, devEmailResponse{
		ID: e.ID, To: e.To, Subject: e.Subject,
		HTML: e.HTML, Text: e.Text, CreatedAt: e.CreatedAt,
	})
}

func (s *Server) handleDeleteAllDevEmails(w http.ResponseWriter, r *http.Request) {
	if !s.devInboxAvailable() {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error":   "not_found",
			"message": "Dev inbox is only available when email.provider is set to 'dev'",
		})
		return
	}
	if err := s.Store.DeleteAllDevEmails(r.Context()); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
