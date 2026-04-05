package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// Server holds dependencies for the HTTP API.
type Server struct {
	Store          storage.Store
	Config         *config.Config
	Router         chi.Router
	SessionManager *auth.SessionManager
}

// NewServer creates a new API server with all routes mounted.
func NewServer(store storage.Store, cfg *config.Config) *Server {
	sessionLifetime := cfg.Auth.SessionLifetimeDuration()
	sm := auth.NewSessionManager(store, cfg.Server.Secret, sessionLifetime)

	s := &Server{
		Store:          store,
		Config:         cfg,
		SessionManager: sm,
	}

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
			r.Post("/check", notImplemented)

			// OAuth
			r.Route("/oauth", func(r chi.Router) {
				r.Get("/{provider}", notImplemented)
				r.Get("/{provider}/callback", notImplemented)
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
				r.Post("/send", notImplemented)
				r.Get("/verify", notImplemented)
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
			r.Post("/", notImplemented)
			r.Get("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Put("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
			r.Post("/{id}/permissions", notImplemented)
			r.Delete("/{id}/permissions/{pid}", notImplemented)
		})

		// Permissions (admin)
		r.Route("/permissions", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Post("/", notImplemented)
			r.Get("/", notImplemented)
		})

		// Users (admin)
		r.Route("/users", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Get("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
			r.Post("/{id}/roles", notImplemented)
			r.Delete("/{id}/roles/{rid}", notImplemented)
			r.Get("/{id}/roles", notImplemented)
			r.Get("/{id}/permissions", notImplemented)
			r.Get("/{id}/audit-logs", notImplemented)
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
			r.Post("/", notImplemented)
			r.Get("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Patch("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
			r.Post("/{id}/rotate", notImplemented)
		})

		// Audit Logs (admin)
		r.Route("/audit-logs", func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Get("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Post("/export", notImplemented)
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
