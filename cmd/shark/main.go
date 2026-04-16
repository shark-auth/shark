package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/api"
	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	configPath := flag.String("config", "sharkauth.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Validate server secret (must be at least 32 bytes for AES-256)
	if len(cfg.Server.Secret) < 32 {
		slog.Error("server.secret must be at least 32 characters", "length", len(cfg.Server.Secret))
		os.Exit(1)
	}

	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.Storage.Path)
	if dataDir != "" && dataDir != "." {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			slog.Error("failed to create data directory", "path", dataDir, "error", err)
			os.Exit(1)
		}
	}

	// Open SQLite database
	store, err := storage.NewSQLiteStore(cfg.Storage.Path)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Run migrations
	slog.Info("running database migrations")
	if err := storage.RunMigrations(store.DB(), migrationsFS, "migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations complete")

	// Bootstrap admin API key on first run
	adminCount, err := store.CountActiveAPIKeysByScope(context.Background(), "*")
	if err != nil {
		slog.Error("failed to check admin keys", "error", err)
		os.Exit(1)
	}
	if adminCount == 0 {
		fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
		if err != nil {
			slog.Error("failed to generate admin key", "error", err)
			os.Exit(1)
		}
		id, _ := gonanoid.New()
		now := time.Now().UTC().Format(time.RFC3339)
		adminKey := &storage.APIKey{
			ID:        "key_" + id,
			Name:      "default-admin",
			KeyHash:   keyHash,
			KeyPrefix: keyPrefix,
			KeySuffix: keySuffix,
			Scopes:    `["*"]`,
			RateLimit: 0,
			CreatedAt: now,
		}
		if err := store.CreateAPIKey(context.Background(), adminKey); err != nil {
			slog.Error("failed to create admin key", "error", err)
			os.Exit(1)
		}
		fmt.Println()
		fmt.Println("  ADMIN API KEY (shown once — save it now)")
		fmt.Println()
		fmt.Printf("    %s\n", fullKey)
		fmt.Println()
		fmt.Println("  Use as: Authorization: Bearer <key>")
		fmt.Println()
	}

	// Create email sender for magic links
	// Use Resend HTTP API when host is smtp.resend.com (SMTP ports blocked on most PaaS)
	var emailSender email.Sender
	if cfg.SMTP.Host == "smtp.resend.com" {
		slog.Info("using Resend HTTP API for email delivery")
		emailSender = email.NewResendSender(cfg.SMTP)
	} else {
		slog.Info("using SMTP for email delivery")
		emailSender = email.NewSMTPSender(cfg.SMTP)
	}

	// Create API server
	srv := api.NewServer(store, cfg, api.WithEmailSender(emailSender))

	// Create HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		slog.Info("SharkAuth starting", "addr", addr)
		slog.Info("admin dashboard", "url", fmt.Sprintf("http://localhost:%d/admin", cfg.Server.Port))
		slog.Info("health check", "url", fmt.Sprintf("http://localhost:%d/healthz", cfg.Server.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
