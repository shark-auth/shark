// Package server wires config, storage, migrations, and HTTP into a runnable unit.
// Extracted so cobra subcommands (serve, dev-mode serve, tests) can share the same path.
package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/api"
	"github.com/sharkauth/sharkauth/internal/auth"
	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/oauth"
	"github.com/sharkauth/sharkauth/internal/proxy"
	"github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/webhook"
)

// Options controls how Run assembles the server. Fields not set receive defaults.
type Options struct {
	ConfigPath    string
	MigrationsFS  embed.FS
	MigrationsDir string

	// DevMode enables ephemeral developer ergonomics: in-db email capture,
	// relaxed CORS, auto-generated secret if missing, and the /admin/dev/* routes.
	DevMode bool

	// SecretOverride, when non-empty, replaces cfg.Server.Secret after Load.
	// Used by --dev to auto-generate a 32-byte secret.
	SecretOverride string

	// StoragePathOverride, when non-empty, replaces cfg.Storage.Path. Used by
	// --dev to point at ./dev.db regardless of the YAML config.
	StoragePathOverride string

	// EmailSenderOverride, when non-nil, replaces the sender chosen from cfg.SMTP.
	// Used by tests to inject MemoryEmailSender.
	EmailSenderOverride email.Sender

	// ProxyUpstream, when non-empty, enables embedded proxy mode: the
	// reverse proxy is mounted as a catch-all AFTER every other route, and
	// cfg.Proxy is overridden to {Enabled: true, Upstream: <this>} with
	// rules and other knobs still coming from the YAML config. Set via the
	// `shark serve --proxy-upstream URL` flag.
	ProxyUpstream string
}

// Bootstrap builds all components (config, store, migrations, admin key, API server)
// without binding to the network. Returned shutdown closes the store.
type Bootstrap struct {
	Config     *config.Config
	Store      *storage.SQLiteStore
	API        *api.Server
	Dispatcher *webhook.Dispatcher
	AdminKey   string // populated only when a new admin key was generated

	// ProxyListeners holds the W15 multi-listener set. One entry per
	// proxy.listeners[] config block with a non-empty bind. Legacy
	// main-port mode uses the catch-all mount inside api.Server and
	// does not populate this slice.
	ProxyListeners []*proxy.Listener
}

// Close releases resources.
func (b *Bootstrap) Close() error {
	if b.Dispatcher != nil {
		b.Dispatcher.Stop()
	}
	if b.Store != nil {
		return b.Store.Close()
	}
	return nil
}

// Build loads config, opens storage, runs migrations, bootstraps admin key, wires API.
// It does not start listening. Callers (cobra subcommands or tests) drive http.Server.
func Build(ctx context.Context, opts Options) (*Bootstrap, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// W15a: warn when legacy proxy fields coexist with the new listeners
	// block. Resolve() already decided listeners win; surfacing the warning
	// here (not inside config/) keeps the config package logger-free while
	// giving operators a clear signal their legacy config is being ignored.
	if cfg.Proxy.Enabled && cfg.Proxy.Upstream != "" && len(cfg.Proxy.Listeners) > 0 {
		// Dual-config is only a real conflict when the legacy fields don't
		// match a Resolve()-synthesised implicit listener. When Listeners was
		// originally empty, Resolve() synthesises one from the legacy fields
		// — that case is not a dual config even though all three are now set.
		// Detect the "user explicitly set both" case by checking that at
		// least one listener has a non-empty Bind (i.e. listeners was set by
		// the user, not synthesised).
		userAuthoredListeners := false
		for _, l := range cfg.Proxy.Listeners {
			if l.Bind != "" {
				userAuthoredListeners = true
				break
			}
		}
		if userAuthoredListeners {
			slog.Warn("proxy: both legacy and listeners fields set, listeners win")
		}
	}

	if opts.SecretOverride != "" {
		cfg.Server.Secret = opts.SecretOverride
	}
	if len(cfg.Server.Secret) < 32 {
		return nil, fmt.Errorf("server.secret must be at least 32 characters (got %d)", len(cfg.Server.Secret))
	}

	// --proxy-upstream override. Keeps rules + other ProxyConfig fields
	// from YAML but flips Enabled on and pins the upstream URL so a single
	// CLI flag is all an operator needs to demo embedded mode.
	if opts.ProxyUpstream != "" {
		cfg.Proxy.Enabled = true
		cfg.Proxy.Upstream = opts.ProxyUpstream
		// Drop any pre-existing listener set — the CLI flag is the
		// single source of truth for the demo path, and Resolve() below
		// will re-synthesise a single implicit main-port listener.
		cfg.Proxy.Listeners = nil
		cfg.Proxy.Resolve()
	}

	cfg.Server.DevMode = opts.DevMode

	// Phase 2: enforce the 3 minimum YAML vars in production. --dev bypasses
	// because it generates its own secret, uses dev.db, and captures email
	// via the dev inbox.
	if !opts.DevMode {
		if cfg.Server.BaseURL == "" {
			return nil, fmt.Errorf("server.base_url is required (phase 2 minimum config)")
		}
		if cfg.Email.Provider == "" && cfg.SMTP.Host == "" {
			return nil, fmt.Errorf("email is not configured: set email.provider + email.api_key, or legacy smtp.*, or run with --dev")
		}
	}

	if opts.StoragePathOverride != "" {
		cfg.Storage.Path = opts.StoragePathOverride
	}
	if opts.DevMode {
		// Wide-open CORS is the standard dev ergonomic — never runs in prod
		// because --dev is only set via the CLI flag, not YAML.
		if len(cfg.Server.CORSOrigins) == 0 {
			cfg.Server.CORSOrigins = []string{"*"}
		}
	}

	dataDir := filepath.Dir(cfg.Storage.Path)
	if dataDir != "" && dataDir != "." {
		// 0o700 (owner rwx only) — the dir holds sharkauth.db + WAL/SHM which
		// contain session material, field-encrypted OAuth tokens, etc. World
		// or group read is never appropriate.
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			return nil, fmt.Errorf("create data dir %q: %w", dataDir, err)
		}
	}

	store, err := storage.NewSQLiteStore(cfg.Storage.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	slog.Info("running database migrations")
	if err := storage.RunMigrations(store.DB(), opts.MigrationsFS, opts.MigrationsDir); err != nil {
		store.Close() //#nosec G104 -- cleanup after migration failure; primary error returned
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	slog.Info("migrations complete")

	// Seed global default roles (admin + member with wildcard perm). Idempotent.
	seedCtx, seedCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer seedCancel()
	rbacMgr := rbac.NewRBACManager(store)
	if err := rbacMgr.SeedDefaultRoles(seedCtx); err != nil {
		slog.Warn("rbac: failed to seed default roles", "error", err)
		// Non-fatal: server continues; roles will be missing until next boot fixes it.
	}

	// Seed editable copies for hosted email templates. Idempotent via INSERT OR
	// IGNORE — only fills rows that don't already exist, so operator edits stick.
	if err := store.SeedEmailTemplates(seedCtx); err != nil {
		slog.Warn("email: failed to seed email templates", "error", err)
		// Non-fatal: server continues; renderer falls back to bundled defaults.
	}

	adminKey, err := bootstrapAdminKey(ctx, store)
	if err != nil {
		store.Close() //#nosec G104 -- cleanup after bootstrap failure; primary error returned
		return nil, fmt.Errorf("bootstrap admin key: %w", err)
	}

	sender := opts.EmailSenderOverride
	if sender == nil {
		if opts.DevMode {
			slog.Info("dev mode: using in-db dev inbox for email capture")
			sender = email.NewDevInboxSender(store)
		} else {
			sender = chooseEmailSender(cfg)
		}
	}

	dispatcher := webhook.New(store)
	dispatcher.Start(ctx)
	// Prune delivery log beyond retention window (default 90d if unset).
	retention := 90 * 24 * time.Hour
	dispatcher.StartRetention(ctx, retention, time.Hour)

	// Instantiate JWT manager and ensure an active signing key exists.
	jwtMgr := jwtpkg.NewManager(&cfg.Auth.JWT, store, cfg.Server.BaseURL, cfg.Server.Secret)
	ensureCtx, ensureCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ensureCancel()
	if err := jwtMgr.EnsureActiveKey(ensureCtx); err != nil {
		slog.Warn("jwt: failed to ensure active signing key", "error", err)
		// Non-fatal: server runs without JWT if key provisioning fails.
	}

	seedCtx2, seedCancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer seedCancel2()
	if err := seedDefaultApplication(seedCtx2, store, cfg); err != nil {
		store.Close() //#nosec G104 -- cleanup after seed failure; primary error returned
		return nil, fmt.Errorf("seed default application: %w", err)
	}

	// OAuth 2.1 server
	var oauthSrv *oauth.Server
	if cfg.OAuthServer.Enabled {
		oauthSrv, err = oauth.NewServer(store, cfg)
		if err != nil {
			slog.Warn("oauth: failed to initialize", "error", err)
			// Non-fatal: server runs without OAuth if initialization fails.
		}
	}

	apiSrv := api.NewServer(store, cfg,
		api.WithEmailSender(sender),
		api.WithWebhookDispatcher(dispatcher),
		api.WithJWTManager(jwtMgr),
		api.WithOAuthServer(oauthSrv),
	)

	// W15: build multi-listener proxy set. cfg.Proxy.Resolve() has already
	// been called in config.Load; entries with empty Bind are legacy
	// main-port mounts handled by the api router's catch-all and skipped
	// here. Building (not binding) happens up front so config errors —
	// missing upstream, bad rule glob — fail at Build time alongside the
	// rest of the startup-critical checks.
	proxyListeners, err := buildProxyListeners(cfg, apiSrv)
	if err != nil {
		store.Close() //#nosec G104 -- cleanup after listener build failure
		return nil, fmt.Errorf("build proxy listeners: %w", err)
	}

	return &Bootstrap{
		Config:         cfg,
		Store:          store,
		API:            apiSrv,
		Dispatcher:     dispatcher,
		AdminKey:       adminKey,
		ProxyListeners: proxyListeners,
	}, nil
}

// buildProxyListeners constructs one *proxy.Listener per config entry with
// a non-empty Bind. Legacy mount-on-main-port entries (Bind=="") are
// skipped — api.Server.initProxy handles them via the router catch-all.
func buildProxyListeners(cfg *config.Config, apiSrv *api.Server) ([]*proxy.Listener, error) {
	out := make([]*proxy.Listener, 0, len(cfg.Proxy.Listeners))
	for i, lc := range cfg.Proxy.Listeners {
		if lc.Bind == "" {
			// Legacy main-port mount — handled inside api.Server.
			continue
		}
		if lc.Upstream == "" {
			return nil, fmt.Errorf("listeners[%d] %s: upstream is required", i, lc.Bind)
		}

		ruleSpecs := make([]proxy.RuleSpec, len(lc.Rules))
		for j, pr := range lc.Rules {
			ruleSpecs[j] = proxy.RuleSpec{
				Path:    pr.Path,
				Methods: pr.Methods,
				Require: pr.Require,
				Allow:   pr.Allow,
				Scopes:  pr.Scopes,
			}
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

		params := proxy.ListenerParams{
			Bind:           lc.Bind,
			Upstream:       lc.Upstream,
			Timeout:        lc.TimeoutDuration(),
			TrustedHeaders: lc.TrustedHeaders,
			StripIncoming:  lc.StripIncomingOrDefault(),
			Rules:          ruleSpecs,
			BreakerConfig:  breakerCfg,
			Logger:         slog.Default(),
		}

		l, err := proxy.NewListener(params)
		if err != nil {
			return nil, fmt.Errorf("listeners[%d]: %w", i, err)
		}
		// Wire auth middleware using the listener's own breaker so a dead
		// upstream on one listener can't tank sessions on another. Using
		// the SetAuthWrap setter avoids a second NewListener call (which
		// would recompile rules and allocate a second Breaker).
		if err := l.SetAuthWrap(apiSrv.ProxyAuthMiddlewareFor(l.Breaker())); err != nil {
			return nil, fmt.Errorf("listeners[%d]: %w", i, err)
		}
		out = append(out, l)
	}
	return out, nil
}

// Serve runs Build then binds the HTTP listener. Blocks until ctx is cancelled.
func Serve(ctx context.Context, opts Options) error {
	b, err := Build(ctx, opts)
	if err != nil {
		return err
	}
	defer b.Close()

	if b.AdminKey != "" {
		printAdminKey(b.AdminKey)
	}

	// T15: mint a one-time bootstrap token if no admin has ever acted on
	// this install. Prints a URL the operator can click to log in without
	// pasting the sk_live_ key. Best-effort — failures just skip the print.
	if tok, err := b.API.MintBootstrapToken(ctx); err != nil {
		slog.Warn("bootstrap token: mint failed", "err", err)
	} else if tok != "" {
		printBootstrapURL(b.Config.Server.Port, tok)
	}

	addr := fmt.Sprintf(":%d", b.Config.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      b.API.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// W15: start each proxy listener BEFORE the main server so a port-
	// in-use failure is surfaced synchronously as a fatal startup error.
	// Partial bind is never acceptable — if any listener can't bind, the
	// entire server exits non-zero so the operator can fix config and
	// retry rather than run half-protected.
	for _, pl := range b.ProxyListeners {
		if err := pl.Start(ctx); err != nil {
			// Best-effort drain any listeners that already bound.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			for _, x := range b.ProxyListeners {
				_ = x.Shutdown(shutdownCtx) //#nosec G104 -- cleanup after fatal bind failure
			}
			cancel()
			return fmt.Errorf("proxy listener %s: %w", pl.Bind, err)
		}
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("SharkAuth starting", "addr", addr, "dev_mode", opts.DevMode)
		slog.Info("admin dashboard", "url", fmt.Sprintf("http://localhost:%d/admin", b.Config.Server.Port))
		slog.Info("health check", "url", fmt.Sprintf("http://localhost:%d/healthz", b.Config.Server.Port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// Fan-out shutdown to proxy listeners first so they stop
		// accepting new connections before the main API server starts
		// draining (upstream auth calls still work while listeners
		// drain).
		for _, pl := range b.ProxyListeners {
			if err := pl.Shutdown(shutdownCtx); err != nil {
				slog.Warn("proxy listener shutdown", "bind", pl.Bind, "err", err)
			}
		}
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		slog.Info("server stopped")
		return nil
	case err, ok := <-errCh:
		if ok && err != nil {
			return err
		}
		return nil
	}
}

func bootstrapAdminKey(ctx context.Context, store *storage.SQLiteStore) (string, error) {
	count, err := store.CountActiveAPIKeysByScope(ctx, "*")
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", nil
	}

	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		return "", err
	}
	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	k := &storage.APIKey{
		ID:        "key_" + id,
		Name:      "default-admin",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeySuffix: keySuffix,
		Scopes:    `["*"]`,
		RateLimit: 0,
		CreatedAt: now,
	}
	if err := store.CreateAPIKey(ctx, k); err != nil {
		return "", err
	}
	return fullKey, nil
}

// chooseEmailSender dispatches on cfg.Email.Provider. The legacy smtp: block is
// still read by config.Load via EmailConfig.Resolve() so existing deployments
// don't need to edit their YAML.
func chooseEmailSender(cfg *config.Config) email.Sender {
	switch cfg.Email.Provider {
	case config.EmailProviderResend:
		slog.Info("using Resend HTTP API for email delivery")
		resendCfg := cfg.SMTP
		resendCfg.Host = "smtp.resend.com"
		if cfg.Email.APIKey != "" {
			resendCfg.Password = cfg.Email.APIKey
		}
		if cfg.Email.From != "" {
			resendCfg.From = cfg.Email.From
		}
		if cfg.Email.FromName != "" {
			resendCfg.FromName = cfg.Email.FromName
		}
		return email.NewResendSender(resendCfg)
	case config.EmailProviderShark:
		slog.Info("using shark.email relay (scaffolding — not yet live)")
		return email.NewSharkEmailSender(cfg.Email.APIKey, cfg.Email.From, cfg.Email.FromName)
	case config.EmailProviderSMTP:
		slog.Info("using SMTP for email delivery")
		smtpCfg := cfg.SMTP
		if cfg.Email.Host != "" {
			smtpCfg.Host = cfg.Email.Host
			smtpCfg.Port = cfg.Email.Port
			smtpCfg.Username = cfg.Email.Username
			smtpCfg.Password = cfg.Email.Password
			smtpCfg.From = cfg.Email.From
			smtpCfg.FromName = cfg.Email.FromName
		}
		return email.NewSMTPSender(smtpCfg)
	default:
		slog.Warn("no email provider configured — using legacy SMTP dispatch", "host", cfg.SMTP.Host)
		if cfg.SMTP.Host == "smtp.resend.com" {
			return email.NewResendSender(cfg.SMTP)
		}
		return email.NewSMTPSender(cfg.SMTP)
	}
}

func printAdminKey(key string) {
	fmt.Println()
	fmt.Println("  ADMIN API KEY (shown once — save it now)")
	fmt.Println()
	fmt.Printf("    %s\n", key)
	fmt.Println()
	fmt.Println("  Use as: Authorization: Bearer <key>")
	fmt.Println()
}

// printBootstrapURL prints a clickable dashboard URL that auto-logs-in via
// the T15 bootstrap token. Only called when no admin has ever acted on this
// install. The token is single-use and expires in 10 minutes.
func printBootstrapURL(port int, token string) {
	fmt.Println()
	fmt.Printf("  Open http://localhost:%d/admin/?bootstrap=%s  (expires in 10 minutes)\n", port, token)
	fmt.Println()
}
