package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// applicationResponse is the JSON shape returned by all app endpoints.
// ClientSecretHash is never included. client_secret is only present at create/rotate time.
type applicationResponse struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	ClientID            string         `json:"client_id"`
	ClientSecretPrefix  string         `json:"client_secret_prefix"`
	AllowedCallbackURLs []string       `json:"allowed_callback_urls"`
	AllowedLogoutURLs   []string       `json:"allowed_logout_urls"`
	AllowedOrigins      []string       `json:"allowed_origins"`
	IsDefault           bool           `json:"is_default"`
	Metadata            map[string]any `json:"metadata"`
	CreatedAt           string         `json:"created_at"`
	UpdatedAt           string         `json:"updated_at"`
}

type applicationResponseWithSecret struct {
	applicationResponse
	ClientSecret string `json:"client_secret"`
}

func appToResponse(a *storage.Application) applicationResponse {
	return applicationResponse{
		ID:                  a.ID,
		Name:                a.Name,
		ClientID:            a.ClientID,
		ClientSecretPrefix:  a.ClientSecretPrefix,
		AllowedCallbackURLs: a.AllowedCallbackURLs,
		AllowedLogoutURLs:   a.AllowedLogoutURLs,
		AllowedOrigins:      a.AllowedOrigins,
		IsDefault:           a.IsDefault,
		Metadata:            a.Metadata,
		CreatedAt:           a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:           a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// validateAppURL parses a URL and enforces http/https scheme.
func validateAppURL(raw string) error {
	if raw == "" {
		return errors.New("url must not be empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("url is not valid: " + err.Error())
	}
	switch u.Scheme {
	case "http", "https":
		// allowed
	case "":
		return errors.New("url must have a scheme (http or https)")
	default:
		// Allow custom mobile schemes (e.g. myapp://) but reject dangerous ones.
		switch u.Scheme {
		case "javascript", "file", "data", "vbscript":
			return errors.New("url scheme not allowed: " + u.Scheme)
		}
	}
	return nil
}

func validateAppURLs(urls []string) error {
	for _, u := range urls {
		if err := validateAppURL(u); err != nil {
			return errors.New("invalid URL " + u + ": " + err.Error())
		}
	}
	return nil
}

// generateAppSecret produces a base62-encoded 32-byte random secret.
func generateAppSecret() (secret, secretHash, secretPrefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	secret = apiBase62Encode(b)
	h := sha256.Sum256([]byte(secret))
	secretHash = hex.EncodeToString(h[:])
	secretPrefix = secret
	if len(secretPrefix) > 8 {
		secretPrefix = secretPrefix[:8]
	}
	return
}

const apiBase62Alpha = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func apiBase62Encode(b []byte) string {
	num := make([]byte, len(b))
	copy(num, b)
	var result []byte
	for !apiIsZero(num) {
		rem := apiDivmod(num, 62)
		result = append(result, apiBase62Alpha[rem])
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) == 0 {
		return "0"
	}
	return string(result)
}

func apiIsZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func apiDivmod(n []byte, d byte) byte {
	var rem uint64
	for i := range n {
		cur := rem*256 + uint64(n[i])
		n[i] = byte(cur / uint64(d))
		rem = cur % uint64(d)
	}
	return byte(rem)
}

// getAppByIDOrClientID tries GetApplicationByID then GetApplicationByClientID.
func (s *Server) getAppByIDOrClientID(r *http.Request, idParam string) (*storage.Application, error) {
	app, err := s.Store.GetApplicationByID(r.Context(), idParam)
	if err == nil {
		return app, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return s.Store.GetApplicationByClientID(r.Context(), idParam)
}

// auditApp logs an application-related audit event.
func (s *Server) auditApp(r *http.Request, action, targetID string) {
	if s.AuditLogger == nil {
		return
	}
	_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
		ActorType:  "admin",
		Action:     action,
		TargetType: "application",
		TargetID:   targetID,
		IP:         r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		Status:     "success",
	})
}

// POST /api/v1/admin/apps
func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string   `json:"name"`
		Callbacks []string `json:"allowed_callback_urls"`
		Logouts   []string `json:"allowed_logout_urls"`
		Origins   []string `json:"allowed_origins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "name is required"))
		return
	}
	if err := validateAppURLs(req.Callbacks); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
		return
	}
	if err := validateAppURLs(req.Logouts); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
		return
	}
	if err := validateAppURLs(req.Origins); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
		return
	}

	nid, err := gonanoid.New(21)
	if err != nil {
		internal(w, err)
		return
	}
	clientID := "shark_app_" + nid

	secret, secretHash, secretPrefix, err := generateAppSecret()
	if err != nil {
		internal(w, err)
		return
	}

	appNid, _ := gonanoid.New()
	now := time.Now().UTC()

	callbacks := req.Callbacks
	if callbacks == nil {
		callbacks = []string{}
	}
	logouts := req.Logouts
	if logouts == nil {
		logouts = []string{}
	}
	origins := req.Origins
	if origins == nil {
		origins = []string{}
	}

	app := &storage.Application{
		ID:                  "app_" + appNid,
		Name:                req.Name,
		ClientID:            clientID,
		ClientSecretHash:    secretHash,
		ClientSecretPrefix:  secretPrefix,
		AllowedCallbackURLs: callbacks,
		AllowedLogoutURLs:   logouts,
		AllowedOrigins:      origins,
		IsDefault:           false,
		Metadata:            map[string]any{},
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.Store.CreateApplication(r.Context(), app); err != nil {
		internal(w, err)
		return
	}

	s.auditApp(r, "app.create", app.ID)

	writeJSON(w, http.StatusCreated, applicationResponseWithSecret{
		applicationResponse: appToResponse(app),
		ClientSecret:        secret,
	})
}

// GET /api/v1/admin/apps
func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}

	apps, err := s.Store.ListApplications(r.Context(), limit, offset)
	if err != nil {
		internal(w, err)
		return
	}

	out := make([]applicationResponse, 0, len(apps))
	for _, a := range apps {
		out = append(out, appToResponse(a))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

// GET /api/v1/admin/apps/{id}
func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	app, err := s.getAppByIDOrClientID(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Application not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appToResponse(app))
}

// PATCH /api/v1/admin/apps/{id}
func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	app, err := s.getAppByIDOrClientID(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Application not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	var req struct {
		Name      *string   `json:"name"`
		Callbacks *[]string `json:"allowed_callback_urls"`
		Logouts   *[]string `json:"allowed_logout_urls"`
		Origins   *[]string `json:"allowed_origins"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.Callbacks != nil {
		if err := validateAppURLs(*req.Callbacks); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
			return
		}
		app.AllowedCallbackURLs = *req.Callbacks
	}
	if req.Logouts != nil {
		if err := validateAppURLs(*req.Logouts); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
			return
		}
		app.AllowedLogoutURLs = *req.Logouts
	}
	if req.Origins != nil {
		if err := validateAppURLs(*req.Origins); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
			return
		}
		app.AllowedOrigins = *req.Origins
	}

	if err := s.Store.UpdateApplication(r.Context(), app); err != nil {
		internal(w, err)
		return
	}

	s.auditApp(r, "app.update", app.ID)

	// Re-fetch to pick up updated_at set by the DB.
	updated, err := s.Store.GetApplicationByID(r.Context(), app.ID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appToResponse(updated))
}

// DELETE /api/v1/admin/apps/{id}
func (s *Server) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	app, err := s.getAppByIDOrClientID(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Application not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if app.IsDefault {
		writeJSON(w, http.StatusConflict, errPayload("cannot_delete_default", "Cannot delete the default application"))
		return
	}

	if err := s.Store.DeleteApplication(r.Context(), app.ID); err != nil {
		internal(w, err)
		return
	}

	s.auditApp(r, "app.delete", app.ID)
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/admin/apps/{id}/rotate-secret
func (s *Server) handleRotateAppSecret(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	app, err := s.getAppByIDOrClientID(r, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Application not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	secret, secretHash, secretPrefix, err := generateAppSecret()
	if err != nil {
		internal(w, err)
		return
	}

	if err := s.Store.RotateApplicationSecret(r.Context(), app.ID, secretHash, secretPrefix); err != nil {
		internal(w, err)
		return
	}

	s.auditApp(r, "app.secret.rotate", app.ID)

	// Re-fetch to get fresh timestamps.
	updated, err := s.Store.GetApplicationByID(r.Context(), app.ID)
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, applicationResponseWithSecret{
		applicationResponse: appToResponse(updated),
		ClientSecret:        secret,
	})
}
