package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sharkauth/sharkauth/internal/admin"
	"github.com/sharkauth/sharkauth/internal/audit"
	"github.com/sharkauth/sharkauth/internal/auth"
	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/authflow"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/oauth"
	"github.com/sharkauth/sharkauth/internal/proxy"
	rbacpkg "github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/sso"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/vault"
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
	OAuthServer       *oauth.Server
	VaultManager      *vault.Manager
	// FlowEngine runs admin-configured auth flows at signup/login/etc.
	// trigger points. nil-safe at the call sites so tests that don't need
	// flows can skip initialisation entirely.
	FlowEngine *authflow.Engine
	// ProxyEngine holds the compiled rule set. nil when proxy is disabled.
	ProxyEngine *proxy.Engine
	// ProxyBreaker drives the circuit breaker + session cache. nil when
	// proxy is disabled; started on NewServer and expected to be stopped
	// by the caller via Shutdown when the server exits.
	ProxyBreaker *proxy.Breaker
	// ProxyHandler is the reverse proxy itself, mounted as a catch-all
	// AFTER every other API route. nil when proxy is disabled so the
	// catch-all block in NewServer short-circuits.
	ProxyHandler *proxy.ReverseProxy

	magicLinkRL *magicLinkRateLimiter
	startTime   time.Time
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

// WithOAuthServer wires the OAuth 2.1 authorization server.
func WithOAuthServer(os *oauth.Server) ServerOption {
	return func(s *Server) {
		s.OAuthServer = os
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
		VaultManager:   vault.NewManager(store, fe),
		startTime:      time.Now().UTC(),
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

	// Phase 6 F3: Initialize the auth flow engine so the signup/login/reset
	// hooks below have somewhere to dispatch. Safe to wire unconditionally —
	// Execute returns Continue when no flows are configured.
	s.FlowEngine = authflow.NewEngine(store, slog.Default())

	// Initialize Audit logger
	s.AuditLogger = audit.NewLogger(store)

	// Initialize SSO manager + handlers
	ssoManager := sso.NewSSOManager(store, sm, cfg)
	s.SSOHandlers = NewSSOHandlers(ssoManager)
	s.SSOHandlers.server = s

	// Initialize proxy (engine + breaker + handler) when enabled. Keeping
	// this before chi.NewRouter keeps the wiring block linear: build the
	// proxy, then mount admin routes that reference it, then mount the
	// catch-all at the very end so it only captures paths no other route
	// claimed.
	s.initProxy()

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

	// RFC 8414 OAuth Authorization Server Metadata (MCP discovery entrypoint)
	r.Get("/.well-known/oauth-authorization-server", oauth.MetadataHandler(cfg.Server.BaseURL))

	// Branding asset serve (A6). Public, content-addressed, immutable-cached.
	// Mounted at root scope — NOT under /api/v1 or /admin — so logos can be
	// embedded in outbound emails and external sites without auth. Must sit
	// before the proxy catch-all at the bottom so chi's trie routes /assets/*
	// here rather than forwarding it upstream.
	r.Get("/assets/branding/*", s.handleBrandingAsset)

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

			// Flow-step MFA verify (public — called during paused flow before session exists)
			r.Route("/flow", func(r chi.Router) {
				r.Post("/mfa/verify", s.handleFlowMFAVerify)
			})

			// SSO auto-route
			r.Get("/sso", s.SSOHandlers.SSOAutoRoute)

			// Self-service session management
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
				r.Get("/sessions", s.handleListMySessions)
				r.Delete("/sessions/{id}", s.handleRevokeMySession)
			})

			// Consent management (session auth - user manages their own consents)
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
				r.Get("/consents", s.handleListConsents)
				r.Delete("/consents/{id}", s.handleRevokeConsent)
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
			r.Delete("/{id}", s.handleDeletePermission)
			// Reverse lookup — which roles/users have this permission?
			r.Get("/{id}/roles", s.handleListRolesByPermission)
			r.Get("/{id}/users", s.handleListUsersByPermission)
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
			r.Get("/{id}/oauth-accounts", s.handleListUserOAuthAccounts)
			r.Delete("/{id}/oauth-accounts/{oauthId}", s.handleDeleteUserOAuthAccount)
			r.Get("/{id}/passkeys", s.handleAdminListUserPasskeys)
			// Admin MFA disable: clears the user's TOTP without their code so
			// support can recover an account when the device is lost. The
			// user-facing /auth/mfa endpoint still requires a current code.
			r.Delete("/{id}/mfa", s.handleAdminDisableUserMFA)
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
			// /events must come before /{id} so chi's trie routes it correctly.
			r.Get("/events", s.handleListWebhookEvents)
			r.Get("/{id}", s.handleGetWebhook)
			r.Patch("/{id}", s.handleUpdateWebhook)
			r.Delete("/{id}", s.handleDeleteWebhook)
			r.Post("/{id}/test", s.handleTestWebhook)
			r.Get("/{id}/deliveries", s.handleListDeliveries)
			r.Post("/{id}/deliveries/{deliveryId}/replay", s.handleReplayWebhookDelivery)
		})

		// Agents (admin)
		r.Route("/agents", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Post("/", s.handleCreateAgent)
			r.Get("/", s.handleListAgents)
			r.Get("/{id}", s.handleGetAgent)
			r.Patch("/{id}", s.handleUpdateAgent)
			r.Delete("/{id}", s.handleDeleteAgent)
			r.Get("/{id}/tokens", s.handleListAgentTokens)
			r.Post("/{id}/tokens/revoke-all", s.handleRevokeAgentTokens)
			r.Post("/{id}/rotate-secret", s.handleAgentRotateSecret)
			r.Get("/{id}/audit", s.handleAgentAuditLogs)
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

		// Vault — third-party OAuth token storage.
		// Admin routes manage the provider catalog + credential rotation; user
		// routes run the connect/disconnect flow with a session cookie; the
		// per-user token retrieval endpoint accepts an OAuth 2.1 bearer from
		// an agent acting on the user's behalf.
		r.Route("/vault", func(r chi.Router) {
			// Admin provider CRUD + template discovery + cross-user connections.
			r.Group(func(r chi.Router) {
				r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
				r.Post("/providers", s.handleCreateVaultProvider)
				r.Get("/providers", s.handleListVaultProviders)
				r.Get("/providers/{id}", s.handleGetVaultProvider)
				r.Patch("/providers/{id}", s.handleUpdateVaultProvider)
				r.Delete("/providers/{id}", s.handleDeleteVaultProvider)
				r.Get("/templates", s.handleListVaultTemplates)
			})

			// User-facing connect flow + connection management (session auth).
			r.Group(func(r chi.Router) {
				r.Use(mw.RequireSessionFunc(sm, s.JWTManager))
				r.Get("/connect/{provider}", s.handleVaultConnectStart)
				r.Get("/callback/{provider}", s.handleVaultCallback)
				r.Get("/connections", s.handleListVaultConnections)
				r.Delete("/connections/{id}", s.handleDeleteVaultConnection)
			})

			// Agent token retrieval (OAuth bearer). Must come AFTER the static
			// prefixes above — chi's trie prefers exact matches, so "providers",
			// "templates", "connect", "callback", "connections" all win the
			// route race before this wildcard is considered.
			r.Get("/{provider}/token", s.handleVaultGetToken)
		})

		// Bootstrap token consume (T15) — NO auth middleware. The token in the
		// request body IS the credential; the handler validates it against an
		// in-memory single-use hash minted at startup. Mounted before the
		// /admin group below so AdminAPIKeyFromStore doesn't gate it.
		r.Post("/admin/bootstrap/consume", s.handleBootstrapConsume)

		// Admin (stats + sessions + dev-mode inbox)
		r.Route("/admin", func(r chi.Router) {
			r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
			r.Get("/health", s.handleAdminHealth)
			r.Get("/config", s.handleAdminConfig)
			r.Get("/stats", s.handleAdminStats)
			r.Get("/stats/trends", s.handleAdminStatsTrends)
			r.Get("/sessions", s.handleAdminListSessions)
			r.Delete("/sessions", s.handleAdminRevokeAllSessions)
			r.Delete("/sessions/{id}", s.handleAdminDeleteSession)
			r.Post("/sessions/purge-expired", s.handlePurgeExpiredSessions)
			r.Post("/audit-logs/purge", s.handlePurgeAuditLogs)
			r.Post("/test-email", s.handleAdminTestEmail)
			r.Get("/email-preview/{template}", s.handleAdminEmailPreview)
			r.Post("/auth/rotate-signing-key", s.handleAdminRotateSigningKey)

			// Cross-user vault connections (admin scope). The /vault/connections
			// endpoint above is session-scoped to the calling user.
			// Accepts optional ?provider_id=<id> to filter by provider.
			r.Get("/vault/connections", s.handleAdminListVaultConnections)
			r.Delete("/vault/connections/{id}", s.handleAdminDeleteVaultConnection)

			// Batch permission usage — replaces 2×N per-row API calls from
			// the PermissionsTab with a single request. Kept in /admin group
			// so the same AdminAPIKeyFromStore middleware gate applies.
			r.Get("/permissions/batch-usage", s.handlePermissionsBatchUsage)

			// Cross-user OAuth consents (admin scope). The /auth/consents
			// endpoint is session-scoped; the dashboard needs a tenant view.
			r.Get("/oauth/consents", s.handleAdminListConsents)
			r.Delete("/oauth/consents/{id}", s.handleAdminRevokeConsent)

			// Admin device-code queue + override decision endpoints. Used by
			// the dashboard to triage pending device flows when the user
			// can't reach the verify URL themselves.
			r.Get("/oauth/device-codes", s.handleAdminListDeviceCodes)
			r.Post("/oauth/device-codes/{user_code}/approve", s.handleAdminApproveDeviceCode)
			r.Post("/oauth/device-codes/{user_code}/deny", s.handleAdminDenyDeviceCode)

			// Admin organization endpoints (admin key auth, not session auth).
			// User-facing /api/v1/organizations/{id} requires a session cookie
			// + RBAC permissions; the dashboard uses admin-key auth, so we
			// duplicate the CRUD surface here under /admin/organizations to
			// avoid muddying the per-user permission model. Mirrors the
			// /admin/sessions, /admin/apps, /admin/flows pattern.
			// Admin user creation (T04). Other /users/* admin routes live under
			// the /api/v1/users group (admin-key auth) above — this endpoint
			// is mounted under /admin to match the dashboard's admin-key URL
			// convention and keep create distinct from the list/patch flow.
			r.Post("/users", s.handleAdminCreateUser)

			r.Post("/organizations", s.handleAdminCreateOrganization)
			r.Get("/organizations", s.handleAdminListOrganizations)
			r.Get("/organizations/{id}", s.handleAdminGetOrganization)
			r.Patch("/organizations/{id}", s.handleAdminUpdateOrganization)
			r.Delete("/organizations/{id}", s.handleAdminDeleteOrganization)
			r.Get("/organizations/{id}/members", s.handleAdminListOrgMembers)
			r.Delete("/organizations/{id}/members/{uid}", s.handleAdminRemoveOrgMember)
			r.Get("/organizations/{id}/roles", s.handleAdminListOrgRoles)
			r.Post("/organizations/{id}/roles", s.handleAdminCreateOrgRole)
			r.Get("/organizations/{id}/invitations", s.handleAdminListOrgInvitations)
			r.Delete("/organizations/{id}/invitations/{invitationId}", s.handleAdminDeleteOrgInvitation)
			r.Post("/organizations/{id}/invitations/{invitationId}/resend", s.handleAdminResendOrgInvitation)

			// Branding admin CRUD + logo upload/delete (Phase A, task A5).
			// Inherits the AdminAPIKeyFromStore middleware from the parent
			// /admin group; asset-serve route /assets/branding/* is mounted
			// separately (A6) since it's public + content-addressed.
			r.Route("/branding", func(r chi.Router) {
				r.Get("/", s.handleGetBranding)
				r.Patch("/", s.handlePatchBranding)
				r.Post("/logo", s.handleUploadLogo)
				r.Delete("/logo", s.handleDeleteLogo)
			})

			// Email template admin CRUD + preview/send-test/reset (Phase A, task A7).
			// Inherits the AdminAPIKeyFromStore middleware from the parent /admin group.
			r.Route("/email-templates", func(r chi.Router) {
				r.Get("/", s.handleListEmailTemplates)
				r.Get("/{id}", s.handleGetEmailTemplate)
				r.Patch("/{id}", s.handlePatchEmailTemplate)
				r.Post("/{id}/preview", s.handlePreviewEmailTemplate)
				r.Post("/{id}/send-test", s.handleSendTestEmail)
				r.Post("/{id}/reset", s.handleResetEmailTemplate)
			})

			if cfg.Server.DevMode {
				r.Get("/dev/emails", s.handleListDevEmails)
				r.Get("/dev/emails/{id}", s.handleGetDevEmail)
				r.Delete("/dev/emails", s.handleDeleteAllDevEmails)
			}

			// Phase 6 P4: proxy admin APIs. Always registered (they 404
			// themselves when the proxy is disabled) so the dashboard can
			// probe and fall back without inspecting the config first.
			r.Get("/proxy/status", s.handleProxyStatus)
			r.Get("/proxy/status/stream", s.handleProxyStatusStream)
			r.Get("/proxy/rules", s.handleProxyRules)
			r.Post("/proxy/simulate", s.handleProxySimulate)

			// Phase 6.6 / Wave D — DB-backed proxy rule overrides. Always
			// available regardless of proxy enable state so admins can
			// stage rules before flipping the proxy on; mutations refresh
			// the live engine via Engine.SetRules when the engine exists.
			r.Get("/proxy/rules/db", s.handleListProxyRules)
			r.Post("/proxy/rules/db", s.handleCreateProxyRule)
			r.Get("/proxy/rules/db/{id}", s.handleGetProxyRule)
			r.Patch("/proxy/rules/db/{id}", s.handleUpdateProxyRule)
			r.Delete("/proxy/rules/db/{id}", s.handleDeleteProxyRule)

			// Phase 6 F3: Auth flow CRUD + dry-run + history. Mounted under
			// the admin group so admin-key auth gates everything.
			r.Route("/flows", func(r chi.Router) {
				r.Post("/", s.handleCreateFlow)
				r.Get("/", s.handleListFlows)
				r.Get("/{id}", s.handleGetFlow)
				r.Patch("/{id}", s.handleUpdateFlow)
				r.Delete("/{id}", s.handleDeleteFlow)
				r.Post("/{id}/test", s.handleTestFlow)
				r.Get("/{id}/runs", s.handleListFlowRuns)
			})
		})
	})

	// OAuth 2.1 endpoints (fosite-backed).
	if s.OAuthServer != nil {
		r.Route("/oauth", func(r chi.Router) {
			r.Post("/token", s.OAuthServer.HandleToken)
			// Authorize endpoints: optionally wrap with session middleware so
			// the user identity is available when a session cookie is present.
			r.Group(func(r chi.Router) {
				r.Use(mw.OptionalSessionFunc(sm, s.JWTManager))
				r.Get("/authorize", s.OAuthServer.HandleAuthorize)
				r.Post("/authorize", s.OAuthServer.HandleAuthorizeDecision)
			})
			// Dynamic Client Registration (RFC 7591 + RFC 7592).
			r.Post("/register", s.OAuthServer.HandleDCRRegister)
			r.Get("/register/{client_id}", s.OAuthServer.HandleDCRGet)
			r.Put("/register/{client_id}", s.OAuthServer.HandleDCRUpdate)
			r.Delete("/register/{client_id}", s.OAuthServer.HandleDCRDelete)
			// Token Introspection (RFC 7662) and Revocation (RFC 7009)
			r.Post("/introspect", s.OAuthServer.HandleIntrospect)
			r.Post("/revoke", s.OAuthServer.HandleRevoke)
			// Device Authorization Grant (RFC 8628)
			r.Post("/device", s.OAuthServer.HandleDeviceAuthorization)
			r.Group(func(r chi.Router) {
				r.Use(mw.OptionalSessionFunc(sm, s.JWTManager))
				r.Get("/device/verify", s.OAuthServer.HandleDeviceVerify)
				r.Post("/device/verify", s.OAuthServer.HandleDeviceApprove)
			})
		})
	}

	// Admin dashboard (Phase 4) — embedded HTML/React bundle, SPA fallback.
	r.Handle("/admin", http.RedirectHandler("/admin/", http.StatusMovedPermanently))
	r.Handle("/admin/*", http.StripPrefix("/admin/", admin.Handler()))

	// Phase 6 P4: reverse proxy catch-all. Mounted last so every /api/v1/*,
	// /oauth/*, /admin/*, /.well-known/* route above wins via chi's trie
	// precedence. Only active when the proxy is configured — when
	// ProxyHandler is nil we skip the handler entirely, keeping the router
	// in its exact pre-P4 shape for deployments that don't use the proxy.
	if s.ProxyHandler != nil {
		r.Handle("/*", s.proxyAuthMiddleware(s.ProxyHandler))
	}

	s.Router = r
	return s
}

// initProxy builds the compiled rules engine, starts the circuit breaker,
// and constructs the ReverseProxy handler. Called from NewServer before
// routes are mounted; a nil-safe no-op when ProxyConfig.Enabled is false
// so test configs (and pre-P4 deployments) skip the setup entirely.
//
// Failures during compilation are surfaced as panics. The proxy's rule
// list lives in user YAML, so a bad rule is a config error the operator
// should fix on next boot — starting the server in a partly-wired state
// would hide the problem behind silent 404s.
func (s *Server) initProxy() {
	cfg := s.Config
	if cfg == nil || !cfg.Proxy.Enabled || cfg.Proxy.Upstream == "" {
		return
	}

	logger := slog.Default()

	ruleSpecs := make([]proxy.RuleSpec, len(cfg.Proxy.Rules))
	for i, pr := range cfg.Proxy.Rules {
		ruleSpecs[i] = proxy.RuleSpec{
			Path:    pr.Path,
			Methods: pr.Methods,
			Require: pr.Require,
			Allow:   pr.Allow,
			Scopes:  pr.Scopes,
		}
	}
	engine, err := proxy.NewEngine(ruleSpecs)
	if err != nil {
		panic("proxy: compiling rules: " + err.Error())
	}
	s.ProxyEngine = engine

	// Wave D: layer DB-backed override rules on top of the YAML bootstrap.
	// Best-effort — if the table doesn't exist yet (first run before the
	// migration applied) or load fails, the proxy keeps the YAML-only set
	// and we surface the error in logs rather than panicking.
	if err := s.refreshProxyEngineFromDB(context.Background()); err != nil {
		logger.Warn("proxy: failed to load DB rule overrides", "err", err)
	}

	breakerCfg := proxy.BreakerConfig{
		HealthURL:        cfg.Server.BaseURL + "/api/v1/admin/health",
		HealthInterval:   10 * time.Second,
		FailureThreshold: 3,
		CacheSize:        10000,
		CacheTTL:         5 * time.Minute,
		NegativeTTL:      30 * time.Second,
		MissBehavior:     proxy.MissReject,
	}
	s.ProxyBreaker = proxy.NewBreaker(breakerCfg, logger)
	// TODO(P4.1): wire the breaker shutdown into server.Serve's shutdown
	// path so SIGTERM stops the monitor cleanly. For now the background
	// goroutine exits when the process does.
	s.ProxyBreaker.Start(context.Background())

	handlerCfg := proxy.Config{
		Enabled:        true,
		Upstream:       cfg.Proxy.Upstream,
		Timeout:        cfg.Proxy.TimeoutDuration(),
		TrustedHeaders: cfg.Proxy.TrustedHeaders,
		StripIncoming:  cfg.Proxy.StripIncomingOrDefault(),
	}
	h, err := proxy.New(handlerCfg, engine, logger)
	if err != nil {
		panic("proxy: building reverse proxy: " + err.Error())
	}
	s.ProxyHandler = h
}

// proxyAuthMiddleware resolves the inbound request's identity (via
// BreakerResolver composing JWT + live-session resolvers) and stashes it
// on the request context so ReverseProxy.ServeHTTP can read it. On any
// resolve error we treat the request as anonymous — the rules engine
// will deny if the matched rule requires authentication, and the proxy
// will translate that into a 401 (unauthenticated) or 403 (authenticated-
// but-unauthorized) based on the resolved identity.
func (s *Server) proxyAuthMiddleware(next http.Handler) http.Handler {
	return s.ProxyAuthMiddlewareFor(s.ProxyBreaker)(next)
}

// ProxyAuthMiddlewareFor returns a middleware that resolves the request's
// identity via the given per-listener breaker. Exposed so the W15
// multi-listener path in internal/server can share the same JWT + session
// resolution code as the legacy main-port mount.
func (s *Server) ProxyAuthMiddlewareFor(breaker *proxy.Breaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		composite := &proxy.BreakerResolver{
			Breaker: breaker,
			JWTResolver: &JWTResolver{
				JWT:   s.JWTManager,
				Store: s.Store,
			},
			Live: &LiveResolver{
				Sessions: s.SessionManager,
				Store:    s.Store,
				RBAC:     s.RBAC,
			},
			Logger: slog.Default(),
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := composite.Resolve(r)
			if err != nil {
				// Downgrade to anonymous — the rules engine is the single
				// source of truth for whether that's allowed for this path.
				// Logging here is intentionally at Debug: a flood of failed
				// resolves (e.g. during an auth outage) shouldn't spam the
				// info log.
				slog.Debug("proxy auth resolve failed, treating as anonymous",
					"err", err,
					"path", r.URL.Path,
				)
				id = proxy.Identity{AuthMethod: "anonymous"}
			}
			r = r.WithContext(proxy.WithIdentity(r.Context(), id))
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	// Ping the database to verify readiness
	if err := s.Store.DB().PingContext(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": "database unreachable"}) //#nosec G104 -- write to ResponseWriter; no actionable recovery
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) //#nosec G104 -- write to ResponseWriter; no actionable recovery
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{ //#nosec G104 -- write to ResponseWriter; no actionable recovery
		"error":   "not_implemented",
		"message": "This endpoint is not yet implemented",
	})
}
