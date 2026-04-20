package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// adminHealthResponse is the response shape for GET /admin/health.
//
// The shape is consumed by the dashboard Overview page health card
// (admin/src/components/overview.tsx, mapHealth). When changing fields
// here, update both. Smoke section 25 asserts the contract.
type adminHealthResponse struct {
	Version        string            `json:"version"`
	UptimeSeconds  int64             `json:"uptime_seconds"`
	DB             healthDBSection   `json:"db"`
	Migrations     healthMigrations  `json:"migrations"`
	JWT            healthJWTSection  `json:"jwt"`
	SMTP           healthSMTPSection `json:"smtp"`
	OAuthProviders []string          `json:"oauth_providers"`
	SSOConnections int               `json:"sso_connections"`
}

// healthDBSection reports database driver + size + status.
type healthDBSection struct {
	Driver string  `json:"driver"`
	SizeMB float64 `json:"size_mb"`
	Status string  `json:"status"`
}

// healthMigrations reports the current applied migration version.
// Name is best-effort and may be empty when not derivable.
type healthMigrations struct {
	Current int64  `json:"current"`
	Name    string `json:"name,omitempty"`
}

// healthJWTSection reports JWT signing mode + key info.
// Mode is "session" (legacy/opaque) or "jwt".
type healthJWTSection struct {
	Mode       string `json:"mode"`
	Algorithm  string `json:"algorithm,omitempty"`
	ActiveKeys int    `json:"active_keys"`
}

// healthSMTPSection reports email provider info. SentToday/DailyLimit are
// nullable because backend does not currently track per-day send counts;
// frontend tolerates null and shows "—".
type healthSMTPSection struct {
	Host       string `json:"host,omitempty"`
	Tier       string `json:"tier"`
	Configured bool   `json:"configured"`
	SentToday  *int   `json:"sent_today"`
	DailyLimit *int   `json:"daily_limit"`
}

// adminConfigSummary holds non-sensitive config fields exposed by GET /admin/config.
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

// currentMigrationVersion returns the highest version_id from goose's tracking
// table (goose_db_version). Returns 0 if the table is missing or empty.
func currentMigrationVersion(ctx context.Context, db *sql.DB) int64 {
	var v sql.NullInt64
	err := db.QueryRowContext(ctx,
		`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1`).Scan(&v)
	if err != nil || !v.Valid {
		return 0
	}
	return v.Int64
}

// handleAdminHealth handles GET /api/v1/admin/health.
// Returns system diagnostics in the nested shape consumed by the dashboard
// Overview health card (admin/src/components/overview.tsx mapHealth).
func (s *Server) handleAdminHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := s.Config

	// --- DB ---
	bytes := dbSizeBytes(ctx, s.Store.DB())
	sizeMB := float64(bytes) / (1024.0 * 1024.0)
	// Round to 2 decimals.
	sizeMB = float64(int64(sizeMB*100+0.5)) / 100.0
	dbStatus := "ok"
	if err := s.Store.DB().PingContext(ctx); err != nil {
		dbStatus = "error"
	}

	// --- Migrations ---
	migCurrent := currentMigrationVersion(ctx, s.Store.DB())

	// --- JWT ---
	jwtMode := "session"
	if cfg.Auth.JWT.Enabled && cfg.Auth.JWT.Mode != "session" {
		jwtMode = "jwt"
	}
	jwtAlg := ""
	jwtKeys := 0
	if keys, err := s.Store.ListJWKSCandidates(ctx, true, time.Time{}); err == nil {
		jwtKeys = len(keys)
		if len(keys) > 0 {
			jwtAlg = keys[0].Algorithm
		}
	}

	// --- SMTP ---
	smtpConfigured := cfg.Email.Provider != "" &&
		cfg.Email.Provider != "dev" &&
		(cfg.Email.Host != "" || cfg.Email.APIKey != "")
	smtpTier := "dev"
	if smtpConfigured {
		smtpTier = "production"
	}

	// --- OAuth providers (reuse logic via config summary helper) ---
	summary := s.buildConfigSummary(ctx)

	resp := adminHealthResponse{
		Version:       resolveAppVersion(),
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		DB: healthDBSection{
			Driver: "sqlite",
			SizeMB: sizeMB,
			Status: dbStatus,
		},
		Migrations: healthMigrations{
			Current: migCurrent,
		},
		JWT: healthJWTSection{
			Mode:       jwtMode,
			Algorithm:  jwtAlg,
			ActiveKeys: jwtKeys,
		},
		SMTP: healthSMTPSection{
			Host:       cfg.Email.Host,
			Tier:       smtpTier,
			Configured: smtpConfigured,
			// sent_today/daily_limit not tracked yet — return null so
			// the frontend renders "—" instead of a fabricated number.
			SentToday:  nil,
			DailyLimit: nil,
		},
		OAuthProviders: summary.OAuthProviders,
		SSOConnections: summary.SSOConnectionsCount,
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

// handleAdminDisableUserMFA handles DELETE /api/v1/users/{id}/mfa.
// Admin-only escape hatch when a user has lost their TOTP device — clears
// the secret, recovery codes, and the enabled flag without requiring the
// user's current TOTP code (the user-facing /auth/mfa endpoint requires it).
// Audited as `admin.mfa.disabled` so the action is traceable.
func (s *Server) handleAdminDisableUserMFA(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	user, err := s.Store.GetUserByID(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "User not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	user.MFAEnabled = false
	user.MFAVerified = false
	user.MFASecret = nil
	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Store.UpdateUser(r.Context(), user); err != nil {
		internal(w, err)
		return
	}
	// Best-effort: clear recovery codes too. A failure here doesn't roll back
	// the disable — admin already wanted MFA off.
	_ = s.Store.DeleteAllMFARecoveryCodesByUserID(r.Context(), userID)

	if s.AuditLogger != nil {
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			Action:     "admin.mfa.disabled",
			TargetType: "user",
			TargetID:   userID,
			IP:         ipOf(r),
			UserAgent:  uaOf(r),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"mfa_enabled": false,
		"user_id":     userID,
	})
}

// handleAdminEmailPreview handles GET /api/v1/admin/email-preview/{template}.
// Renders the requested transactional email template against canned sample
// data and returns the resulting HTML so the dashboard can show a preview
// pane on the Authentication page.
//
// Supported templates: magic_link, verify_email, password_reset,
// organization_invitation. Unknown templates return 404.
func (s *Server) handleAdminEmailPreview(w http.ResponseWriter, r *http.Request) {
	tpl := chi.URLParam(r, "template")
	appName := "SharkAuth"
	if s.Config != nil && s.Config.Server.BaseURL != "" {
		// Display name only — no real URL exposure needed in preview copy.
		appName = "SharkAuth"
	}

	var html string
	var err error
	switch tpl {
	case "magic_link":
		html, err = email.RenderMagicLink(email.MagicLinkData{
			AppName:       appName,
			MagicLinkURL:  "https://example.com/auth/magic?token=preview-token-not-real",
			ExpiryMinutes: 15,
		})
	case "verify_email":
		html, err = email.RenderVerifyEmail(email.VerifyEmailData{
			AppName:       appName,
			VerifyURL:     "https://example.com/auth/verify?token=preview-token-not-real",
			ExpiryMinutes: 60,
		})
	case "password_reset":
		html, err = email.RenderPasswordReset(email.PasswordResetData{
			AppName:       appName,
			ResetURL:      "https://example.com/auth/reset?token=preview-token-not-real",
			ExpiryMinutes: 30,
		})
	case "organization_invitation":
		html, err = email.RenderOrganizationInvitation(email.OrganizationInvitationData{
			AppName:      appName,
			OrgName:      "Acme Corp",
			Role:         "member",
			AcceptURL:    "https://example.com/orgs/acme/accept?token=preview",
			InviterEmail: "alice@acme.com",
			ExpiryHours:  72,
		})
	default:
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Unknown email template: "+tpl))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"template": tpl,
		"html":     html,
	})
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
