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
}

// Bootstrap builds all components (config, store, migrations, admin key, API server)
// without binding to the network. Returned shutdown closes the store.
type Bootstrap struct {
	Config     *config.Config
	Store      *storage.SQLiteStore
	API        *api.Server
	Dispatcher *webhook.Dispatcher
	AdminKey   string // populated only when a new admin key was generated
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

	if opts.SecretOverride != "" {
		cfg.Server.Secret = opts.SecretOverride
	}
	if len(cfg.Server.Secret) < 32 {
		return nil, fmt.Errorf("server.secret must be at least 32 characters (got %d)", len(cfg.Server.Secret))
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

	apiSrv := api.NewServer(store, cfg,
		api.WithEmailSender(sender),
		api.WithWebhookDispatcher(dispatcher),
		api.WithJWTManager(jwtMgr),
	)

	return &Bootstrap{
		Config:     cfg,
		Store:      store,
		API:        apiSrv,
		Dispatcher: dispatcher,
		AdminKey:   adminKey,
	}, nil
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

	addr := fmt.Sprintf(":%d", b.Config.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      b.API.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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
