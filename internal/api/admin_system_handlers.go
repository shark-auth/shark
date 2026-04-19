package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// adminHealthResponse is the response shape for GET /admin/health.
type adminHealthResponse struct {
	Version       string             `json:"version"`
	UptimeSeconds int64              `json:"uptime_seconds"`
	DBSizeBytes   int64              `json:"db_size_bytes"`
	Config        adminConfigSummary `json:"config"`
}

// adminConfigSummary holds non-sensitive config fields exposed by both
// GET /admin/health and GET /admin/config.
type adminConfigSummary struct {
	DevMode             bool     `json:"dev_mode"`
	JWTMode             bool     `json:"jwt_mode"`
	SMTPConfigured      bool     `json:"smtp_configured"`
	OAuthProviders      []string `json:"oauth_providers"`
	SSOConnectionsCount int      `json:"sso_connections_count"`
	BaseURL             string   `json:"base_url"`
	CORSOrigins         []string `json:"cors_origins"`
}

// resolveAppVersion returns the build-time or module version, falling back to "dev".
func resolveAppVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// dbSizeBytes queries SQLite PRAGMA page_count * page_size to get the current
// database file size in bytes. Returns 0 on error so health stays non-fatal.
func dbSizeBytes(ctx context.Context, db *sql.DB) int64 {
	var pageCount, pageSize int64
	if err := db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		return 0
	}
	if err := db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0
	}
	return pageCount * pageSize
}

// buildConfigSummary constructs the non-sensitive config snapshot from the server state.
func (s *Server) buildConfigSummary(ctx context.Context) adminConfigSummary {
	cfg := s.Config

	// Detect configured OAuth providers (mirrors logic in oauth_handlers.go).
	var providers []string
	if cfg.Social.Google.ClientID != "" && cfg.Social.Google.ClientSecret != "" {
		providers = append(providers, "google")
	}
	if cfg.Social.GitHub.ClientID != "" && cfg.Social.GitHub.ClientSecret != "" {
		providers = append(providers, "github")
	}
	if cfg.Social.Apple.ClientID != "" && cfg.Social.Apple.TeamID != "" {
		providers = append(providers, "apple")
	}
	if cfg.Social.Discord.ClientID != "" && cfg.Social.Discord.ClientSecret != "" {
		providers = append(providers, "discord")
	}
	if providers == nil {
		providers = []string{}
	}

	// SMTP is configured when a real provider and credentials are present.
	smtpConfigured := cfg.Email.Provider != "" &&
		cfg.Email.Provider != "dev" &&
		(cfg.Email.Host != "" || cfg.Email.APIKey != "")

	// Count SSO connections (best-effort; 0 on error).
	ssoCount, _ := s.Store.CountSSOConnections(ctx, false)

	cors := cfg.Server.CORSOrigins
	if cors == nil {
		cors = []string{}
	}

	return adminConfigSummary{
		DevMode:             cfg.Server.DevMode,
		JWTMode:             cfg.Auth.JWT.Enabled && cfg.Auth.JWT.Mode != "session",
		SMTPConfigured:      smtpConfigured,
		OAuthProviders:      providers,
		SSOConnectionsCount: ssoCount,
		BaseURL:             cfg.Server.BaseURL,
		CORSOrigins:         cors,
	}
}

// handleAdminHealth handles GET /api/v1/admin/health.
// Returns system diagnostics: version, uptime, DB size, and a sanitised config snapshot.
func (s *Server) handleAdminHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := adminHealthResponse{
		Version:       resolveAppVersion(),
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		DBSizeBytes:   dbSizeBytes(ctx, s.Store.DB()),
		Config:        s.buildConfigSummary(ctx),
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleAdminConfig handles GET /api/v1/admin/config.
// Returns the sanitised runtime configuration (no secrets).
func (s *Server) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.buildConfigSummary(r.Context()))
}

// handleAdminListOrganizations handles GET /api/v1/admin/organizations.
// Lists ALL organizations (admin view, not user-scoped).
func (s *Server) handleAdminListOrganizations(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.Store.ListAllOrganizations(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]organizationResponse, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, orgToResponse(o))
	}
	writeJSON(w, http.StatusOK, map[string]any{"organizations": out})
}

// handleAdminGetOrganization handles GET /api/v1/admin/organizations/{id}.
func (s *Server) handleAdminGetOrganization(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	org, err := s.Store.GetOrganizationByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	}
	writeJSON(w, http.StatusOK, orgToResponse(org))
}

// handleAdminListOrgMembers handles GET /api/v1/admin/organizations/{id}/members.
func (s *Server) handleAdminListOrgMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	members, err := s.Store.ListOrganizationMembers(r.Context(), orgID)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

// handleAdminTestEmail handles POST /api/v1/admin/test-email.
// Sends a test email to verify SMTP/email provider configuration.
func (s *Server) handleAdminTestEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.To == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Field 'to' is required"))
		return
	}

	// Get email sender from MagicLinkManager (or dev inbox in dev mode)
	var sender email.Sender
	if s.MagicLinkManager != nil {
		sender = s.MagicLinkManager.Sender()
	}
	if sender == nil {
		writeJSON(w, http.StatusServiceUnavailable, errPayload("email_not_configured", "No email provider configured"))
		return
	}

	msg := &email.Message{
		To:      req.To,
		Subject: "SharkAuth Test Email",
		HTML:    "<h2>SharkAuth Email Test</h2><p>This is a test email from your SharkAuth instance.</p><p>If you received this, your email configuration is working correctly.</p>",
		Text:    "SharkAuth Email Test\n\nThis is a test email from your SharkAuth instance.\nIf you received this, your email configuration is working correctly.",
	}

	if err := sender.Send(msg); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("send_failed", "Failed to send test email: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"sent": true, "to": req.To})
}

// handleAdminListUserPasskeys handles GET /api/v1/users/{id}/passkeys.
// Lists passkey credentials for a user (admin access).
func (s *Server) handleAdminListUserPasskeys(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	creds, err := s.Store.GetPasskeysByUserID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}
	if creds == nil {
		creds = []*storage.PasskeyCredential{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"passkeys": creds})
}

// handleAdminRotateSigningKey handles POST /api/v1/admin/auth/rotate-signing-key.
// Generates a new JWT signing key and retires the current one.
func (s *Server) handleAdminRotateSigningKey(w http.ResponseWriter, r *http.Request) {
	if s.JWTManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, errPayload("jwt_not_configured", "JWT mode is not enabled"))
		return
	}

	if err := s.JWTManager.GenerateAndStore(r.Context(), true); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("rotation_failed", "Failed to rotate signing key: "+err.Error()))
		return
	}

	// Get the new active key info
	key, err := s.Store.GetActiveSigningKey(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"rotated": true})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rotated":   true,
		"kid":       key.KID,
		"algorithm": key.Algorithm,
	})
}
