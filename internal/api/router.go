package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sharkauth/sharkauth/internal/audit"
	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/storage"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// Server holds dependencies for the HTTP API.
type Server struct {
	Store            storage.Store
	Config           *config.Config
	Router           chi.Router
	SessionManager   *auth.SessionManager
	OAuthManager     *auth.OAuthManager
	MagicLinkManager *auth.MagicLinkManager
	RBAC             *rbac.RBACManager
	AuditLogger      *audit.Logger
	RateLimiter      *auth.TokenBucket
	magicLinkRL      *magicLinkRateLimiter
}

// ServerOption configures optional dependencies for the Server.
type ServerOption func(*Server)

// WithEmailSender enables magic link functionality with the provided email sender.
func WithEmailSender(sender email.Sender) ServerOption {
	return func(s *Server) {
		s.MagicLinkManager = auth.NewMagicLinkManager(s.Store, sender, s.SessionManager, s.Config)
	}
}

// NewServer creates a new API server with all routes mounted.
func NewServer(store storage.Store, cfg *config.Config, opts ...ServerOption) *Server {
	sessionLifetime := cfg.Auth.SessionLifetimeDuration()
	sm := auth.NewSessionManager(store, cfg.Server.Secret, sessionLifetime)

	s := &Server{
		Store:          store,
		Config:         cfg,
		SessionManager: sm,
		magicLinkRL:    newMagicLinkRateLimiter(60 * time.Second),
		RateLimiter:    auth.NewTokenBucket(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Initialize OAuth manager
	s.initOAuthManager()

	// Initialize RBAC manager
	s.RBAC = rbac.NewRBACManager(store)

	// Initialize Audit logger
	s.AuditLogger = audit.NewLogger(store)

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(mw.RateLimit(100, 100)) // 100 req/s burst, 100 tokens

	// Health check
	r.Get("/healthz", s.handleHealthz)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", s.handleSignup)
			r.Post("/login", s.handleLogin)
			r.Post("/logout", s.handleLogout)
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm))
				r.Use(mw.RequireMFA)
				r.Get("/me", s.handleMe)
			})
			r.Post("/check", s.handleAuthCheck)

			// OAuth
			r.Route("/oauth", func(r chi.Router) {
				r.Get("/{provider}", s.handleOAuthStart)
				r.Get("/{provider}/callback", s.handleOAuthCallback)
			})

			// Passkeys
			r.Route("/passkey", func(r chi.Router) {
				r.Post("/register/begin", notImplemented)
				r.Post("/register/finish", notImplemented)
				r.Post("/login/begin", notImplemented)
				r.Post("/login/finish", notImplemented)
				r.Get("/credentials", notImplemented)
				r.Delete("/credentials/{id}", notImplemented)
				r.Patch("/credentials/{id}", notImplemented)
			})

			// Magic Links
			r.Route("/magic-link", func(r chi.Router) {
				r.Post("/send", s.handleMagicLinkSend)
				r.Get("/verify", s.handleMagicLinkVerify)
			})

			// MFA
			r.Route("/mfa", func(r chi.Router) {
				// Challenge and recovery: require session (any, including partial mfa_passed=false)
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm))
					r.Post("/challenge", s.handleMFAChallenge)
					r.Post("/recovery", s.handleMFARecovery)
				})
				// Enroll, verify, disable, recovery-codes: require full session (mfa_passed=true)
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm))
					r.Use(mw.RequireMFA)
					r.Post("/enroll", s.handleMFAEnroll)
					r.Post("/verify", s.handleMFAVerify)
					r.Delete("/", s.handleMFADisable)
					r.Get("/recovery-codes", s.handleMFARecoveryCodes)
				})
			})

			// SSO auto-route
			r.Get("/sso", notImplemented)
		})

		// Roles (admin)
		r.Route("/roles", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Post("/", s.handleCreateRole)
			r.Get("/", s.handleListRoles)
			r.Get("/{id}", s.handleGetRole)
			r.Put("/{id}", s.handleUpdateRole)
			r.Delete("/{id}", s.handleDeleteRole)
			r.Post("/{id}/permissions", s.handleAttachPermission)
			r.Delete("/{id}/permissions/{pid}", s.handleDetachPermission)
		})

		// Permissions (admin)
		r.Route("/permissions", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Post("/", s.handleCreatePermission)
			r.Get("/", s.handleListPermissions)
		})

		// Users (admin)
		r.Route("/users", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Get("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
			r.Post("/{id}/roles", s.handleAssignRole)
			r.Delete("/{id}/roles/{rid}", s.handleRemoveRole)
			r.Get("/{id}/roles", s.handleListUserRoles)
			r.Get("/{id}/permissions", s.handleListUserPermissions)
			r.Get("/{id}/audit-logs", s.handleUserAuditLogs)
		})

		// SSO connections (admin)
		r.Route("/sso", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Route("/connections", func(r chi.Router) {
				r.Post("/", notImplemented)
				r.Get("/", notImplemented)
				r.Get("/{id}", notImplemented)
				r.Put("/{id}", notImplemented)
				r.Delete("/{id}", notImplemented)
			})
			r.Get("/saml/{connection_id}/metadata", notImplemented)
			r.Post("/saml/{connection_id}/acs", notImplemented)
			r.Get("/oidc/{connection_id}/auth", notImplemented)
			r.Get("/oidc/{connection_id}/callback", notImplemented)
		})

		// API Keys (admin)
		r.Route("/api-keys", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Post("/", s.handleCreateAPIKey)
			r.Get("/", s.handleListAPIKeys)
			r.Get("/{id}", s.handleGetAPIKey)
			r.Patch("/{id}", s.handleUpdateAPIKey)
			r.Delete("/{id}", s.handleRevokeAPIKey)
			r.Post("/{id}/rotate", s.handleRotateAPIKey)
		})

		// Audit Logs (admin)
		r.Route("/audit-logs", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Get("/", s.handleListAuditLogs)
			r.Get("/{id}", s.handleGetAuditLog)
			r.Post("/export", s.handleExportAuditLogs)
		})

		// Migration (admin)
		r.Route("/migrate", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Post("/auth0", notImplemented)
			r.Get("/{id}", notImplemented)
		})
	})

	// Admin dashboard (static files)
	r.Handle("/admin/*", http.StripPrefix("/admin/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Placeholder: will serve embedded Svelte dashboard files in Wave 4
		http.Error(w, "Admin dashboard not yet built", http.StatusNotFound)
	})))

	s.Router = r
	return s
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "not_implemented",
		"message": "This endpoint is not yet implemented",
	})
}
