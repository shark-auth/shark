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
	"strings"
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
	Slug                string         `json:"slug"`
	ClientID            string         `json:"client_id"`
	ClientSecretPrefix  string         `json:"client_secret_prefix"`
	AllowedCallbackURLs []string       `json:"allowed_callback_urls"`
	AllowedLogoutURLs   []string       `json:"allowed_logout_urls"`
	AllowedOrigins      []string       `json:"allowed_origins"`
	IsDefault           bool           `json:"is_default"`
	Metadata            map[string]any `json:"metadata"`

	IntegrationMode       string          `json:"integration_mode"`
	BrandingOverride      json.RawMessage `json:"branding_override,omitempty"`
	ProxyLoginFallback    string          `json:"proxy_login_fallback,omitempty"`
	ProxyLoginFallbackURL string          `json:"proxy_login_fallback_url,omitempty"`
	ProxyPublicDomain     string          `json:"proxy_public_domain,omitempty"`
	ProxyProtectedURL     string          `json:"proxy_protected_url,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type applicationResponseWithSecret struct {
	applicationResponse
	ClientSecret string `json:"client_secret"`
}

func appToResponse(a *storage.Application) applicationResponse {
	var override json.RawMessage
	if a.BrandingOverride != "" {
		override = json.RawMessage(a.BrandingOverride)
	}
	return applicationResponse{
		ID:                    a.ID,
		Name:                  a.Name,
		Slug:                  a.Slug,
		ClientID:              a.ClientID,
		ClientSecretPrefix:    a.ClientSecretPrefix,
		AllowedCallbackURLs:   a.AllowedCallbackURLs,
		AllowedLogoutURLs:     a.AllowedLogoutURLs,
		AllowedOrigins:        a.AllowedOrigins,
		IsDefault:             a.IsDefault,
		Metadata:              a.Metadata,
		IntegrationMode:       a.IntegrationMode,
		BrandingOverride:      override,
		ProxyLoginFallback:    a.ProxyLoginFallback,
		ProxyLoginFallbackURL: a.ProxyLoginFallbackURL,
		ProxyPublicDomain:     a.ProxyPublicDomain,
		ProxyProtectedURL:     a.ProxyProtectedURL,
		CreatedAt:             a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// validIntegrationMode checks the enum for integration_mode.
func validIntegrationMode(m string) bool {
	switch m {
	case "hosted", "components", "proxy", "custom":
		return true
	}
	return false
}

// validProxyLoginFallback checks the enum for proxy_login_fallback.
func validProxyLoginFallback(m string) bool {
	switch m {
	case "hosted", "custom_url":
		return true
	}
	return false
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
		n[i] = byte(cur / uint64(d)) //#nosec G115 -- base-62 long division: d is a byte (≤255) and cur/d fits in a byte by construction
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

// POST /api/v1/admin/apps
func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string   `json:"name"`
		Slug              string   `json:"slug"`
		Callbacks         []string `json:"allowed_callback_urls"`
		Logouts           []string `json:"allowed_logout_urls"`
		Origins           []string `json:"allowed_origins"`
		IntegrationMode   string   `json:"integration_mode"`
		ProxyPublicDomain string   `json:"proxy_public_domain"`
		ProxyProtectedURL string   `json:"proxy_protected_url"`
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
	if req.ProxyProtectedURL != "" {
		if err := validateAppURL(req.ProxyProtectedURL); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", "proxy_protected_url: "+err.Error()))
			return
		}
	}

	mode := req.IntegrationMode
	if mode == "" {
		if req.ProxyPublicDomain != "" {
			mode = "proxy"
		} else {
			mode = "custom"
		}
	}
	if !validIntegrationMode(mode) {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "invalid integration_mode"))
		return
	}

	// Resolve slug: auto-generate from name if not provided, validate if provided.
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	} else {
		if err := validateSlug(slug); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_slug", err.Error()))
			return
		}
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
		Slug:                slug,
		ClientID:            clientID,
		ClientSecretHash:    secretHash,
		ClientSecretPrefix:  secretPrefix,
		AllowedCallbackURLs: callbacks,
		AllowedLogoutURLs:   logouts,
		AllowedOrigins:      origins,
		IntegrationMode:     mode,
		ProxyPublicDomain:   req.ProxyPublicDomain,
		ProxyProtectedURL:   req.ProxyProtectedURL,
		IsDefault:           false,
		Metadata:            map[string]any{},
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.Store.CreateApplication(r.Context(), app); err != nil {
		if isDuplicateErr(err) {
			writeJSON(w, http.StatusConflict, errPayload("slug_conflict", "An application with this slug already exists"))
			return
		}
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		metaBytes, _ := json.Marshal(map[string]any{
			"app_name":            app.Name,
			"client_id":           app.ClientID,
			"redirect_uris_count": len(app.AllowedCallbackURLs),
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "app.create",
			TargetType: "application",
			TargetID:   app.ID,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

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
		Name                  *string         `json:"name"`
		Slug                  *string         `json:"slug,omitempty"`
		Callbacks             *[]string       `json:"allowed_callback_urls"`
		Logouts               *[]string       `json:"allowed_logout_urls"`
		Origins               *[]string       `json:"allowed_origins"`
		IntegrationMode       *string         `json:"integration_mode,omitempty"`
		BrandingOverride      json.RawMessage `json:"branding_override,omitempty"`
		ProxyLoginFallback    *string         `json:"proxy_login_fallback,omitempty"`
		ProxyLoginFallbackURL *string         `json:"proxy_login_fallback_url,omitempty"`
		ProxyPublicDomain     *string         `json:"proxy_public_domain"`
		ProxyProtectedURL     *string         `json:"proxy_protected_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	changedFields := []string{}
	if req.Name != nil {
		app.Name = *req.Name
		changedFields = append(changedFields, "name")
	}
	if req.Slug != nil {
		if err := validateSlug(*req.Slug); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_slug", err.Error()))
			return
		}
		app.Slug = *req.Slug
		changedFields = append(changedFields, "slug")
	}
	if req.Callbacks != nil {
		if err := validateAppURLs(*req.Callbacks); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
			return
		}
		app.AllowedCallbackURLs = *req.Callbacks
		changedFields = append(changedFields, "allowed_callback_urls")
	}
	if req.Logouts != nil {
		if err := validateAppURLs(*req.Logouts); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
			return
		}
		app.AllowedLogoutURLs = *req.Logouts
		changedFields = append(changedFields, "allowed_logout_urls")
	}
	if req.Origins != nil {
		if err := validateAppURLs(*req.Origins); err != nil {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
			return
		}
		app.AllowedOrigins = *req.Origins
		changedFields = append(changedFields, "allowed_origins")
	}
	if req.IntegrationMode != nil {
		if !validIntegrationMode(*req.IntegrationMode) {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request",
				"integration_mode must be one of: hosted, components, proxy, custom"))
			return
		}
		app.IntegrationMode = *req.IntegrationMode
		changedFields = append(changedFields, "integration_mode")
	}
	if req.BrandingOverride != nil {
		// Normalise: null or "" clears the override. Otherwise must be a JSON object.
		raw := strings.TrimSpace(string(req.BrandingOverride))
		switch raw {
		case "", "null", `""`:
			app.BrandingOverride = ""
		default:
			var probe map[string]any
			if err := json.Unmarshal([]byte(raw), &probe); err != nil {
				writeJSON(w, http.StatusBadRequest, errPayload("invalid_request",
					"branding_override must be a JSON object"))
				return
			}
			app.BrandingOverride = raw
		}
		changedFields = append(changedFields, "branding_override")
	}
	if req.ProxyLoginFallback != nil {
		if !validProxyLoginFallback(*req.ProxyLoginFallback) {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request",
				"proxy_login_fallback must be one of: hosted, custom_url"))
			return
		}
		app.ProxyLoginFallback = *req.ProxyLoginFallback
		changedFields = append(changedFields, "proxy_login_fallback")
	}
	if req.ProxyLoginFallbackURL != nil {
		if *req.ProxyLoginFallbackURL != "" {
			if err := validateAppURL(*req.ProxyLoginFallbackURL); err != nil {
				writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", err.Error()))
				return
			}
		}
		app.ProxyLoginFallbackURL = *req.ProxyLoginFallbackURL
		changedFields = append(changedFields, "proxy_login_fallback_url")
	}
	if req.ProxyPublicDomain != nil {
		app.ProxyPublicDomain = *req.ProxyPublicDomain
		changedFields = append(changedFields, "proxy_public_domain")
	}
	if req.ProxyProtectedURL != nil {
		if *req.ProxyProtectedURL != "" {
			if err := validateAppURL(*req.ProxyProtectedURL); err != nil {
				writeJSON(w, http.StatusBadRequest, errPayload("invalid_url", "proxy_protected_url: "+err.Error()))
				return
			}
		}
		app.ProxyProtectedURL = *req.ProxyProtectedURL
		changedFields = append(changedFields, "proxy_protected_url")
	}

	if err := s.Store.UpdateApplication(r.Context(), app); err != nil {
		if isDuplicateErr(err) {
			writeJSON(w, http.StatusConflict, errPayload("slug_conflict", "An application with this slug already exists"))
			return
		}
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		metaBytes, _ := json.Marshal(map[string]any{
			"app_id":         app.ID,
			"changed_fields": changedFields,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "app.update",
			TargetType: "application",
			TargetID:   app.ID,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

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

	if s.AuditLogger != nil {
		metaBytes, _ := json.Marshal(map[string]any{
			"app_id":   app.ID,
			"app_name": app.Name,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "app.delete",
			TargetType: "application",
			TargetID:   app.ID,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}
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

	oldHash := app.ClientSecretHash

	if err := s.Store.RotateApplicationSecret(r.Context(), app.ID, secretHash, secretPrefix); err != nil {
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		oldKid := oldHash
		if len(oldKid) > 12 {
			oldKid = oldKid[:12]
		}
		newKid := secretHash
		if len(newKid) > 12 {
			newKid = newKid[:12]
		}
		metaBytes, _ := json.Marshal(map[string]any{
			"app_id":  app.ID,
			"old_kid": oldKid,
			"new_kid": newKid,
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "app.secret.rotate",
			TargetType: "application",
			TargetID:   app.ID,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

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
