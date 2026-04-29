package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/email"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/version"
)

// adminHealthResponse is the response shape for GET /admin/health.
type adminHealthResponse struct {
	Status         string            `json:"status"`
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
type healthMigrations struct {
	Current int64  `json:"current"`
	Name    string `json:"name,omitempty"`
}

// healthJWTSection reports JWT signing mode + key info.
type healthJWTSection struct {
	Mode       string `json:"mode"`
	Algorithm  string `json:"algorithm,omitempty"`
	ActiveKeys int    `json:"active_keys"`
}

// healthSMTPSection reports email provider info.
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
	Environment         string   `json:"environment"`
	BaseURL             string   `json:"base_url"`
	CORSOrigins         []string `json:"cors_origins"`
	SSOConnectionsCount int      `json:"sso_connections_count"`
	OAuthProviders      []string `json:"oauth_providers"`

	Server   adminServerConfig  `json:"server"`
	Auth     adminAuthConfig    `json:"auth"`
	Passkey  adminPasskeyConfig `json:"passkey"`
	Email    adminEmailConfig   `json:"email"`
	Audit    adminAuditConfig   `json:"audit"`
	JWT      adminJWTConfig     `json:"jwt"`
	MagicLink adminMagicLinkConfig `json:"magic_link"`
	PasswordReset adminPasswordResetConfig `json:"password_reset"`
	Social   adminSocialConfig  `json:"social"`
}

type adminServerConfig struct {
	Port        int      `json:"port"`
	BaseURL     string   `json:"base_url"`
	CORSOrigins []string `json:"cors_origins"`
	CORSRelaxed bool     `json:"cors_relaxed"`
	DevMode     bool     `json:"dev_mode"`
}

type adminPasswordResetConfig struct {
	TTL string `json:"ttl"`
}

type adminAuthConfig struct {
	SessionLifetime   string `json:"session_lifetime"`
	PasswordMinLength int    `json:"password_min_length"`
}

type adminAuditConfig struct {
	Retention       string `json:"retention"`
	CleanupInterval string `json:"cleanup_interval"`
}

type adminEmailConfig struct {
	Provider         string `json:"provider"`
	APIKey           string `json:"api_key,omitempty"`
	From             string `json:"from"`
	FromName         string `json:"from_name"`
	PreviousProvider string `json:"previous_provider,omitempty"`
}

type adminPasskeyConfig struct {
	Enabled          bool   `json:"enabled"`
	RPID             string `json:"rp_id"`
	RPName           string `json:"rp_name"`
	Origin           string `json:"origin"`
	UserVerification string `json:"user_verification"`
	Attestation      string `json:"attestation"`
}

type adminJWTConfig struct {
	Enabled    bool   `json:"enabled"`
	Mode       string `json:"mode"`
	Issuer     string `json:"issuer"`
	Audience   string `json:"audience"`
	ClockSkew  string `json:"clock_skew"`
	Algorithm  string `json:"algorithm"`
	Lifetime   string `json:"lifetime"`
	ActiveKeys int    `json:"active_keys"`
}

type adminMagicLinkConfig struct {
	TTL string `json:"ttl"`
}

type adminSocialConfig struct {
	RedirectURL string           `json:"redirect_url"`
	Google      adminOAuthCreds  `json:"google"`
	GitHub      adminOAuthCreds  `json:"github"`
}

type adminOAuthCreds struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"` // only shown if set
}

// resolveAppVersion returns the build-time or module version.
func resolveAppVersion() string {
	return version.Version
}

// dbSizeBytes queries SQLite PRAGMA page_count * page_size.
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

// buildConfigSummary constructs the non-sensitive config snapshot.
func (s *Server) buildConfigSummary(ctx context.Context) adminConfigSummary {
	cfg := s.Config

	var providers []string
	if cfg.Social.Google.ClientID != "" {
		providers = append(providers, "google")
	}
	if cfg.Social.GitHub.ClientID != "" {
		providers = append(providers, "github")
	}
	if providers == nil {
		providers = []string{}
	}

	ssoCount, _ := s.Store.CountSSOConnections(ctx, false)

	cors := cfg.Server.CORSOrigins
	if cors == nil {
		cors = []string{}
	}

	jwtAlg := ""
	jwtKeys := 0
	jwtLifetime := cfg.Auth.JWT.AccessTokenTTL
	if jwtLifetime == "" {
		jwtLifetime = "15m"
	}
	if keys, err := s.Store.ListJWKSCandidates(ctx, true, time.Time{}); err == nil {
		jwtKeys = len(keys)
		if len(keys) > 0 {
			jwtAlg = keys[0].Algorithm
		}
	}
	if jwtAlg == "" {
		jwtAlg = cfg.OAuthServer.SigningAlgorithm
	}
	if jwtAlg == "" {
		jwtAlg = "ES256"
	}

	sessionLifetime := cfg.Auth.SessionLifetime
	if sessionLifetime == "" {
		sessionLifetime = "30d"
	}

	minLength := cfg.Auth.PasswordMinLength
	if minLength == 0 {
		minLength = 8
	}

	return adminConfigSummary{
		DevMode:             cfg.Server.DevMode,
		Environment:         "production",
		BaseURL:             cfg.Server.BaseURL,
		CORSOrigins:         cors,
		SSOConnectionsCount: ssoCount,
		OAuthProviders:      providers,

		Server: adminServerConfig{
			Port:        cfg.Server.Port,
			BaseURL:     cfg.Server.BaseURL,
			CORSOrigins: cors,
			CORSRelaxed: cfg.Server.CORSRelaxed,
			DevMode:     cfg.Server.DevMode,
		},
		Auth: adminAuthConfig{
			SessionLifetime:   sessionLifetime,
			PasswordMinLength: minLength,
		},
		Passkey: adminPasskeyConfig{
			Enabled:          cfg.Passkeys.RPID != "" || cfg.Passkeys.RPName != "",
			RPID:             cfg.Passkeys.RPID,
			RPName:           cfg.Passkeys.RPName,
			Origin:           cfg.Passkeys.Origin,
			UserVerification: cfg.Passkeys.UserVerification,
			Attestation:      cfg.Passkeys.Attestation,
		},
		Email: adminEmailConfig{
			Provider:         cfg.Email.Provider,
			APIKey:           cfg.Email.APIKey,
			From:             cfg.Email.From,
			FromName:         cfg.Email.FromName,
			PreviousProvider: cfg.Email.PreviousProvider,
		},
		Audit: adminAuditConfig{
			Retention:       cfg.Audit.Retention,
			CleanupInterval: cfg.Audit.CleanupInterval,
		},
		JWT: adminJWTConfig{
			Enabled:    cfg.Auth.JWT.Enabled,
			Mode:       cfg.Auth.JWT.Mode,
			Issuer:     cfg.Auth.JWT.Issuer,
			Audience:   cfg.Auth.JWT.Audience,
			ClockSkew:  cfg.Auth.JWT.ClockSkew,
			Algorithm:  jwtAlg,
			Lifetime:   jwtLifetime,
			ActiveKeys: jwtKeys,
		},
		MagicLink: adminMagicLinkConfig{
			TTL: cfg.MagicLink.TokenLifetime,
		},
		PasswordReset: adminPasswordResetConfig{
			TTL: cfg.PasswordReset.TokenLifetime,
		},
		Social: adminSocialConfig{
			RedirectURL: cfg.Social.RedirectURL,
			Google: adminOAuthCreds{
				ClientID:     cfg.Social.Google.ClientID,
				ClientSecret: hideSecret(cfg.Social.Google.ClientSecret),
			},
			GitHub: adminOAuthCreds{
				ClientID:     cfg.Social.GitHub.ClientID,
				ClientSecret: hideSecret(cfg.Social.GitHub.ClientSecret),
			},
		},
	}
}

func hideSecret(s string) string {
	if s == "" {
		return ""
	}
	return "********"
}

// currentMigrationVersion returns the highest version_id.
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
func (s *Server) handleAdminHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg := s.Config

	bytes := dbSizeBytes(ctx, s.Store.DB())
	sizeMB := float64(bytes) / (1024.0 * 1024.0)
	sizeMB = float64(int64(sizeMB*100+0.5)) / 100.0
	dbStatus := "ok"
	if err := s.Store.DB().PingContext(ctx); err != nil {
		dbStatus = "error"
	}

	migCurrent := currentMigrationVersion(ctx, s.Store.DB())

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

	smtpConfigured := cfg.Email.Provider != "" &&
		cfg.Email.Provider != "dev" &&
		(cfg.Email.Host != "" || cfg.Email.APIKey != "")
	smtpTier := "dev"
	if smtpConfigured {
		smtpTier = "production"
	}

	summary := s.buildConfigSummary(ctx)

	resp := adminHealthResponse{
		Status:        "ok",
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
			SentToday:  nil,
			DailyLimit: nil,
		},
		OAuthProviders: summary.OAuthProviders,
		SSOConnections: summary.SSOConnectionsCount,
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleAdminConfig handles GET /api/v1/admin/config.
func (s *Server) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.buildConfigSummary(r.Context()))
}

// handleAdminListOrganizations handles GET /api/v1/admin/organizations.
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
func (s *Server) handleAdminTestEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		To string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.To == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Field 'to' is required"))
		return
	}

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
		HTML:    "<h2>SharkAuth Email Test</h2><p>This is a test email from your SharkAuth instance.</p>",
		Text:    "SharkAuth Email Test\n\nThis is a test email from your SharkAuth instance.",
	}

	if err := sender.Send(msg); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("send_failed", "Failed to send: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"sent": true, "to": req.To})
}

// handleAdminListUserPasskeys handles GET /api/v1/users/{id}/passkeys.
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
	_ = s.Store.DeleteAllMFARecoveryCodesByUserID(r.Context(), userID)

	if s.AuditLogger != nil {
		mfaMeta, _ := json.Marshal(map[string]any{
			"target_user_id": userID,
			"mfa_method":     "totp",
			"reason":         "admin_reset",
			"request_id":     r.Header.Get("X-Request-Id"),
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "admin.mfa.disabled",
			TargetType: "user",
			TargetID:   userID,
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
			Metadata:   string(mfaMeta),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"mfa_enabled": false, "user_id": userID})
}

// handleAdminEmailPreview handles GET /api/v1/admin/email-preview/{template}.
func (s *Server) handleAdminEmailPreview(w http.ResponseWriter, r *http.Request) {
	tpl := chi.URLParam(r, "template")
	appName := "SharkAuth"

	ctx := r.Context()
	branding, _ := s.Store.ResolveBranding(ctx, "")
	var rendered email.Rendered
	var err error
	switch tpl {
	case "magic_link":
		rendered, err = email.RenderMagicLink(ctx, s.Store, branding, email.MagicLinkData{
			AppName:       appName,
			MagicLinkURL:  "https://example.com/auth/magic?token=preview",
			ExpiryMinutes: 15,
		})
	case "verify_email":
		rendered, err = email.RenderVerifyEmail(ctx, s.Store, branding, email.VerifyEmailData{
			AppName:       appName,
			VerifyURL:     "https://example.com/auth/verify?token=preview",
			ExpiryMinutes: 60,
		})
	case "password_reset":
		rendered, err = email.RenderPasswordReset(ctx, s.Store, branding, email.PasswordResetData{
			AppName:       appName,
			ResetURL:      "https://example.com/auth/reset?token=preview",
			ExpiryMinutes: 30,
		})
	case "organization_invitation":
		rendered, err = email.RenderOrganizationInvitation(ctx, s.Store, branding, email.OrganizationInvitationData{
			AppName:      appName,
			OrgName:      "Acme Corp",
			Role:         "member",
			AcceptURL:    "https://example.com/orgs/acme/accept?token=preview",
			InviterEmail: "alice@acme.com",
			ExpiryHours:  72,
		})
	case "welcome":
		rendered, err = email.RenderWelcome(ctx, s.Store, branding, email.WelcomeData{
			AppName:      appName,
			UserEmail:    "user@example.com",
			DashboardURL: "https://example.com/dashboard",
		})
	default:
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Unknown template: "+tpl))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"template": tpl,
		"subject":  rendered.Subject,
		"html":     rendered.HTML,
	})
}

// handleAdminRotateSigningKey handles POST /api/v1/admin/auth/rotate-signing-key.
func (s *Server) handleAdminRotateSigningKey(w http.ResponseWriter, r *http.Request) {
	if s.JWTManager == nil {
		writeJSON(w, http.StatusServiceUnavailable, errPayload("jwt_not_configured", "JWT mode not enabled"))
		return
	}

	if err := s.JWTManager.GenerateAndStore(r.Context(), true); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("rotation_failed", err.Error()))
		return
	}

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

// handleAdminUpdateConfig handles PATCH /api/v1/admin/config.
func (s *Server) handleAdminUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Auth struct {
			SessionLifetime   *string `json:"session_lifetime"`
			PasswordMinLength *int    `json:"password_min_length"`
		} `json:"auth"`
		Passkeys struct {
			RPName           *string `json:"rp_name"`
			RPID             *string `json:"rp_id"`
			UserVerification *string `json:"user_verification"`
		} `json:"passkeys"`
		Email struct {
			Provider         *string `json:"provider"`
			APIKey           *string `json:"api_key"`
			From             *string `json:"from"`
			FromName         *string `json:"from_name"`
			PreviousProvider *string `json:"previous_provider"`
		} `json:"email"`
		Server struct {
			CORSRelaxed *bool `json:"cors_relaxed"`
		} `json:"server"`
		Audit struct {
			Retention       *string `json:"retention"`
			CleanupInterval *string `json:"cleanup_interval"`
		} `json:"audit"`
		Social struct {
			RedirectURL *string `json:"redirect_url"`
			Google      *struct {
				ClientID     *string `json:"client_id"`
				ClientSecret *string `json:"client_secret"`
			} `json:"google"`
			GitHub      *struct {
				ClientID     *string `json:"client_id"`
				ClientSecret *string `json:"client_secret"`
			} `json:"github"`
		} `json:"social"`
		JWT struct {
			Enabled   *bool   `json:"enabled"`
			Mode      *string `json:"mode"`
			Issuer    *string `json:"issuer"`
			Audience  *string `json:"audience"`
			ClockSkew *string `json:"clock_skew"`
		} `json:"jwt"`
		PasswordReset struct {
			TTL *string `json:"ttl"`
		} `json:"password_reset"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	cfg := s.Config
	if req.Auth.SessionLifetime != nil {
		cfg.Auth.SessionLifetime = *req.Auth.SessionLifetime
	}
	if req.Auth.PasswordMinLength != nil {
		cfg.Auth.PasswordMinLength = *req.Auth.PasswordMinLength
	}
	if req.Passkeys.RPName != nil {
		cfg.Passkeys.RPName = *req.Passkeys.RPName
	}
	if req.Passkeys.RPID != nil {
		cfg.Passkeys.RPID = *req.Passkeys.RPID
	}
	if req.Passkeys.UserVerification != nil {
		cfg.Passkeys.UserVerification = *req.Passkeys.UserVerification
	}
	if req.Email.Provider != nil {
		newProvider := *req.Email.Provider
		// When switching to dev, preserve the previous provider so the toggle
		// is reversible. When switching away from dev, restore only if the
		// caller doesn't supply an explicit previous_provider.
		if newProvider == "dev" && cfg.Email.Provider != "dev" {
			cfg.Email.PreviousProvider = cfg.Email.Provider
		}
		cfg.Email.Provider = newProvider
	}
	if req.Email.PreviousProvider != nil {
		cfg.Email.PreviousProvider = *req.Email.PreviousProvider
	}
	if req.Email.APIKey != nil {
		cfg.Email.APIKey = *req.Email.APIKey
	}
	if req.Email.From != nil {
		cfg.Email.From = *req.Email.From
	}
	if req.Email.FromName != nil {
		cfg.Email.FromName = *req.Email.FromName
	}
	if req.Server.CORSRelaxed != nil {
		cfg.Server.CORSRelaxed = *req.Server.CORSRelaxed
	}
	if req.Audit.Retention != nil {
		cfg.Audit.Retention = *req.Audit.Retention
	}
	if req.Audit.CleanupInterval != nil {
		cfg.Audit.CleanupInterval = *req.Audit.CleanupInterval
	}

	// Update Social
	if req.Social.RedirectURL != nil {
		cfg.Social.RedirectURL = *req.Social.RedirectURL
	}
	if req.Social.Google != nil {
		if req.Social.Google.ClientID != nil {
			cfg.Social.Google.ClientID = *req.Social.Google.ClientID
		}
		if req.Social.Google.ClientSecret != nil && *req.Social.Google.ClientSecret != "********" {
			cfg.Social.Google.ClientSecret = *req.Social.Google.ClientSecret
		}
	}
	if req.Social.GitHub != nil {
		if req.Social.GitHub.ClientID != nil {
			cfg.Social.GitHub.ClientID = *req.Social.GitHub.ClientID
		}
		if req.Social.GitHub.ClientSecret != nil && *req.Social.GitHub.ClientSecret != "********" {
			cfg.Social.GitHub.ClientSecret = *req.Social.GitHub.ClientSecret
		}
	}

	// Update JWT
	if req.JWT.Enabled != nil {
		cfg.Auth.JWT.Enabled = *req.JWT.Enabled
	}
	if req.JWT.Mode != nil {
		cfg.Auth.JWT.Mode = *req.JWT.Mode
	}
	if req.JWT.Issuer != nil {
		cfg.Auth.JWT.Issuer = *req.JWT.Issuer
	}
	if req.JWT.Audience != nil {
		cfg.Auth.JWT.Audience = *req.JWT.Audience
	}
	if req.JWT.ClockSkew != nil {
		cfg.Auth.JWT.ClockSkew = *req.JWT.ClockSkew
	}
	if req.PasswordReset.TTL != nil {
		cfg.PasswordReset.TokenLifetime = *req.PasswordReset.TTL
	}

	// W17 Phase A: persist to DB (primary). YAML Save is kept as a secondary
	// write so operators with a yaml file on disk stay in sync until Phase H
	// removes the yaml path entirely.
	if err := config.SaveRuntime(r.Context(), s.Store, cfg); err != nil {
		slog.Error("config: failed to save to db", "error", err)
		writeJSON(w, http.StatusInternalServerError, errPayload("save_failed", "Failed to persist"))
		return
	}

	if s.AuditLogger != nil {
		configMeta, _ := json.Marshal(map[string]any{
			"target_id":    "system_config",
			"changed_by":   "admin_key",
			"email_provider": cfg.Email.Provider,
			"jwt_enabled":  cfg.Auth.JWT.Enabled,
			"request_id":   r.Header.Get("X-Request-Id"),
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			Action:     "admin.config.updated",
			TargetType: "system",
			TargetID:   "system_config",
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Metadata:   string(configMeta),
		})
	}

	// Hot-reload services
	if s.MagicLinkManager != nil && req.Email.Provider != nil {
		var newSender email.Sender
		switch cfg.Email.Provider {
		case "shark", "resend":
			newSender = email.NewResendSender(cfg.SMTP)
		case "smtp":
			newSender = email.NewSMTPSender(cfg.SMTP)
		case "dev":
			newSender = email.NewDevInboxSender(s.Store)
		}
		if newSender != nil {
			s.MagicLinkManager.SetSender(newSender)
			slog.Info("config: email sender re-initialized", "provider", cfg.Email.Provider)
		}
	}

	writeJSON(w, http.StatusOK, s.buildConfigSummary(r.Context()))
}

// handleAdminLogStream streams real-time audit logs via SSE.
func (s *Server) handleAdminLogStream(w http.ResponseWriter, r *http.Request) {
	if s.WebhookDispatcher == nil {
		writeJSON(w, http.StatusServiceUnavailable, errPayload("unavailable", "log streaming unavailable"))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, errPayload("unsupported", "streaming unsupported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.WebhookDispatcher.Subscribe()
	defer s.WebhookDispatcher.Unsubscribe(ch)

	fmt.Fprintf(w, "data: {\"event\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case env := <-ch:
			if env.Event != "system.audit_log" {
				continue
			}
			b, _ := json.Marshal(env.Data)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}
