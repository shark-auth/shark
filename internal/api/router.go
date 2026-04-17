package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sharkauth/sharkauth/internal/audit"
	"github.com/sharkauth/sharkauth/internal/auth"
	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/email"
	rbacpkg "github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/sso"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/webhook"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// Server holds dependencies for the HTTP API.
type Server struct {
	Store            storage.Store
	Config           *config.Config
	Router           chi.Router
	SessionManager   *auth.SessionManager
	PasskeyManager   *auth.PasskeyManager
	OAuthManager     *auth.OAuthManager
	MagicLinkManager *auth.MagicLinkManager
	JWTManager       *jwtpkg.Manager
	RBAC             *rbacpkg.RBACManager
	AuditLogger      *audit.Logger
	RateLimiter      *auth.TokenBucket
	LockoutManager   *auth.LockoutManager
	FieldEncryptor    *auth.FieldEncryptor
	SSOHandlers       *SSOHandlers
	WebhookDispatcher *webhook.Dispatcher
	magicLinkRL       *magicLinkRateLimiter
}

// ServerOption configures optional dependencies for the Server.
type ServerOption func(*Server)

// WithEmailSender enables magic link functionality with the provided email sender.
func WithEmailSender(sender email.Sender) ServerOption {
	return func(s *Server) {
		s.MagicLinkManager = auth.NewMagicLinkManager(s.Store, sender, s.SessionManager, s.Config)
	}
}

// WithWebhookDispatcher wires an already-started dispatcher so emission sites
// can fan out events. Passed from server.Build after it calls Start().
func WithWebhookDispatcher(d *webhook.Dispatcher) ServerOption {
	return func(s *Server) {
		s.WebhookDispatcher = d
	}
}

// WithJWTManager wires a JWT Manager for Bearer token issuance and validation.
func WithJWTManager(jm *jwtpkg.Manager) ServerOption {
	return func(s *Server) {
		s.JWTManager = jm
	}
}

// NewServer creates a new API server with all routes mounted.
func NewServer(store storage.Store, cfg *config.Config, opts ...ServerOption) *Server {
	sessionLifetime := cfg.Auth.SessionLifetimeDuration()
	sm := auth.NewSessionManager(store, cfg.Server.Secret, sessionLifetime, cfg.Server.BaseURL)

	// Initialize field-level encryption
	fe, err := auth.NewFieldEncryptor(cfg.Server.Secret)
	if err != nil {
		// This should not happen since we validate secret length on startup
		panic("failed to initialize field encryption: " + err.Error())
	}

	s := &Server{
		Store:          store,
		Config:         cfg,
		SessionManager: sm,
		magicLinkRL:    newMagicLinkRateLimiter(60 * time.Second),
		RateLimiter:    auth.NewTokenBucket(),
		LockoutManager: auth.NewLockoutManager(5, 15*time.Minute),
		FieldEncryptor: fe,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Initialize Passkey manager
	pm, err := auth.NewPasskeyManager(store, sm, cfg.Passkeys)
	if err != nil {
		// Log but don't fail - passkeys are optional
		// If config is invalid (e.g. no RPID), passkey endpoints will return errors
		pm = nil
	}
	s.PasskeyManager = pm

	// Initialize OAuth manager
	s.initOAuthManager()

	// Initialize RBAC manager
	s.RBAC = rbacpkg.NewRBACManager(store)

	// Initialize Audit logger
	s.AuditLogger = audit.NewLogger(store)

	// Initialize SSO manager + handlers
	ssoManager := sso.NewSSOManager(store, sm, cfg)
	s.SSOHandlers = NewSSOHandlers(ssoManager)
	s.SSOHandlers.server = s

	// Email verification lookup for middleware
	isEmailVerified := func(ctx context.Context, userID string) (bool, error) {
		user, err := store.GetUserByID(ctx, userID)
		if err != nil {
			return false, err
		}
		return user.EmailVerified, nil
	}
	requireVerified := mw.RequireEmailVerifiedFunc(isEmailVerified)

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(mw.MaxBodySize(1 << 20)) // 1 MB request body limit
	r.Use(mw.SecurityHeaders())
	r.Use(mw.RateLimit(100, 100)) // 100 req/s burst, 100 tokens

	// CORS (must be before route handlers)
	if len(cfg.Server.CORSOrigins) > 0 {
		r.Use(mw.CORS(cfg.Server.CORSOrigins))
	}

	// Health check
	r.Get("/healthz", s.handleHealthz)

	// JWKS endpoint (RFC 7517) — public, no auth, top-level
	r.Get("/.well-known/jwks.json", s.HandleJWKS)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", s.handleSignup)
			r.Post("/login", s.handleLogin)
			r.Post("/logout", s.handleLogout)
			// GET /me: allowed without email verification (so frontend can check status)
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
				r.Use(mw.RequireMFA)
				r.Get("/me", s.handleMe)
			})
			// DELETE /me: requires verified email
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
				r.Use(mw.RequireMFA)
				r.Use(requireVerified)
				r.Delete("/me", s.handleDeleteMe)
			})
			// Email verification
			r.Route("/email", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
					r.Post("/verify/send", s.handleEmailVerifySend)
				})
				r.Get("/verify", s.handleEmailVerify)
			})

			// OAuth
			r.Route("/oauth", func(r chi.Router) {
				r.Get("/{provider}", s.handleOAuthStart)
				r.Get("/{provider}/callback", s.handleOAuthCallback)
			})

			// Passkeys
			r.Route("/passkey", func(r chi.Router) {
				// Registration requires auth + verified email
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
					r.Use(requireVerified)
					r.Post("/register/begin", s.handlePasskeyRegisterBegin)
					r.Post("/register/finish", s.handlePasskeyRegisterFinish)
				})
				// Login is public
				r.Post("/login/begin", s.handlePasskeyLoginBegin)
				r.Post("/login/finish", s.handlePasskeyLoginFinish)
				// Credential management requires auth + verified email
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
					r.Use(requireVerified)
					r.Get("/credentials", s.handlePasskeyCredentialsList)
					r.Delete("/credentials/{id}", s.handlePasskeyCredentialDelete)
					r.Patch("/credentials/{id}", s.handlePasskeyCredentialRename)
				})
			})

			// Magic Links
			r.Route("/magic-link", func(r chi.Router) {
				r.Post("/send", s.handleMagicLinkSend)
				r.Get("/verify", s.handleMagicLinkVerify)
			})

			// Password management
			r.Route("/password", func(r chi.Router) {
				// Public: forgot password flow
				r.Post("/send-reset-link", s.handlePasswordResetSend)
				r.Post("/reset", s.handlePasswordReset)
				// Authenticated: change password (requires verified email)
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
					r.Use(mw.RequireMFA)
					r.Use(requireVerified)
					r.Post("/change", s.handleChangePassword)
				})
			})

			// MFA
			r.Route("/mfa", func(r chi.Router) {
				// Challenge and recovery: require session only (login flow, no email check)
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
					r.Post("/challenge", s.handleMFAChallenge)
					r.Post("/recovery", s.handleMFARecovery)
				})
				// Enroll, verify, disable, recovery-codes: require full session + verified email
				r.Group(func(r chi.Router) {
					r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
					r.Use(mw.RequireMFA)
					r.Use(requireVerified)
					r.Post("/enroll", s.handleMFAEnroll)
					r.Post("/verify", s.handleMFAVerify)
					r.Delete("/", s.handleMFADisable)
					r.Get("/recovery-codes", s.handleMFARecoveryCodes)
				})
			})

			// SSO auto-route
			r.Get("/sso", s.SSOHandlers.SSOAutoRoute)

			// Self-service session management
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
				r.Get("/sessions", s.handleListMySessions)
				r.Delete("/sessions/{id}", s.handleRevokeMySession)
			})

			// JWT self-revoke (session-auth, cookie OR JWT)
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
				r.Post("/revoke", s.handleUserRevoke)
			})
		})

		// Organizations (user-facing — session cookie auth)
		r.Route("/organizations", func(r chi.Router) {
			r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
			r.Post("/", s.handleCreateOrganization)
			r.Get("/", s.handleListMyOrganizations)
			r.Post("/invitations/{token}/accept", s.handleAcceptOrgInvitation)

			// Per-org routes — all under {id} for backward compatibility.
			// RequireOrgPermission reads {id} as fallback when {org_id} is absent.
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", s.handleGetOrganization)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "org", "update")).Patch("/", s.handleUpdateOrganization)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "org", "delete")).Delete("/", s.handleDeleteOrganization)
				r.Get("/members", s.handleListOrganizationMembers)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "members", "update_role")).Patch("/members/{uid}", s.handleUpdateOrganizationMemberRole)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "members", "remove")).Delete("/members/{uid}", s.handleRemoveOrganizationMember)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "members", "invite")).Post("/invitations", s.handleCreateOrgInvitation)

				// Org RBAC management routes (org ID comes from parent {id} param).
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "roles", "read")).Get("/roles", s.handleListOrgRoles)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "roles", "create")).Post("/roles", s.handleCreateOrgRole)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "roles", "read")).Get("/roles/{role_id}", s.handleGetOrgRole)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "roles", "create")).Patch("/roles/{role_id}", s.handleUpdateOrgRole)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "roles", "create")).Delete("/roles/{role_id}", s.handleDeleteOrgRole)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "roles", "assign")).Post("/members/{user_id}/roles/{role_id}", s.handleGrantOrgRole)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "roles", "revoke")).Delete("/members/{user_id}/roles/{role_id}", s.handleRevokeOrgRole)
				r.With(rbacpkg.RequireOrgPermission(s.RBAC, "members", "read")).Get("/members/{user_id}/permissions", s.handleGetEffectiveOrgPerms)
			})
		})

		// Roles (admin)
		r.Route("/roles", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
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
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/", s.handleCreatePermission)
			r.Get("/", s.handleListPermissions)
		})

		// Auth check (admin) — validates if a user has a specific permission
		r.Group(func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/auth/check", s.handleAuthCheck)
		})

		// Users (admin)
		r.Route("/users", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Get("/", s.handleListUsers)
			r.Get("/{id}", s.handleGetUser)
			r.Delete("/{id}", s.handleDeleteUser)
			r.Patch("/{id}", s.handleUpdateUser)
			r.Post("/{id}/roles", s.handleAssignRole)
			r.Delete("/{id}/roles/{rid}", s.handleRemoveRole)
			r.Get("/{id}/roles", s.handleListUserRoles)
			r.Get("/{id}/permissions", s.handleListUserPermissions)
			r.Get("/{id}/audit-logs", s.handleUserAuditLogs)
			r.Get("/{id}/sessions", s.handleListUserSessions)
			r.Delete("/{id}/sessions", s.handleRevokeUserSessions)
		})

		// SSO connections (admin + public endpoints)
		r.Route("/sso", func(r chi.Router) {
			// Admin CRUD
			r.Group(func(r chi.Router) {
				r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
				r.Route("/connections", func(r chi.Router) {
					r.Post("/", s.SSOHandlers.CreateConnection)
					r.Get("/", s.SSOHandlers.ListConnections)
					r.Get("/{id}", s.SSOHandlers.GetConnection)
					r.Put("/{id}", s.SSOHandlers.UpdateConnection)
					r.Delete("/{id}", s.SSOHandlers.DeleteConnection)
				})
			})
			// Public SSO endpoints
			r.Get("/saml/{connection_id}/metadata", s.SSOHandlers.SAMLMetadata)
			r.Post("/saml/{connection_id}/acs", s.SSOHandlers.SAMLACS)
			r.Get("/oidc/{connection_id}/auth", s.SSOHandlers.OIDCAuth)
			r.Get("/oidc/{connection_id}/callback", s.SSOHandlers.OIDCCallback)
		})

		// API Keys (admin)
		r.Route("/api-keys", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/", s.handleCreateAPIKey)
			r.Get("/", s.handleListAPIKeys)
			r.Get("/{id}", s.handleGetAPIKey)
			r.Patch("/{id}", s.handleUpdateAPIKey)
			r.Delete("/{id}", s.handleRevokeAPIKey)
			r.Post("/{id}/rotate", s.handleRotateAPIKey)
		})

		// Audit Logs (admin)
		r.Route("/audit-logs", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Get("/", s.handleListAuditLogs)
			r.Get("/{id}", s.handleGetAuditLog)
			r.Post("/export", s.handleExportAuditLogs)
		})

		// Migration (admin)
		r.Route("/migrate", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/auth0", notImplemented)
			r.Get("/{id}", notImplemented)
		})

		// Webhooks (admin)
		r.Route("/webhooks", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/", s.handleCreateWebhook)
			r.Get("/", s.handleListWebhooks)
			r.Get("/{id}", s.handleGetWebhook)
			r.Patch("/{id}", s.handleUpdateWebhook)
			r.Delete("/{id}", s.handleDeleteWebhook)
			r.Post("/{id}/test", s.handleTestWebhook)
			r.Get("/{id}/deliveries", s.handleListDeliveries)
		})

		// Applications (admin)
		r.Route("/admin/apps", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/", s.handleCreateApp)
			r.Get("/", s.handleListApps)
			r.Get("/{id}", s.handleGetApp)
			r.Patch("/{id}", s.handleUpdateApp)
			r.Delete("/{id}", s.handleDeleteApp)
			r.Post("/{id}/rotate-secret", s.handleRotateAppSecret)
		})

		// Admin JWT JTI revocation
		r.Group(func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/admin/auth/revoke-jti", s.handleAdminRevokeJTI)
		})

		// Admin (stats + sessions + dev-mode inbox)
		r.Route("/admin", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Get("/stats", s.handleAdminStats)
			r.Get("/stats/trends", s.handleAdminStatsTrends)
			r.Get("/sessions", s.handleAdminListSessions)
			r.Delete("/sessions/{id}", s.handleAdminDeleteSession)

			if cfg.Server.DevMode {
				r.Get("/dev/emails", s.handleListDevEmails)
				r.Get("/dev/emails/{id}", s.handleGetDevEmail)
				r.Delete("/dev/emails", s.handleDeleteAllDevEmails)
			}
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
	// Ping the database to verify readiness
	if err := s.Store.DB().PingContext(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": "database unreachable"})
		return
	}

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
