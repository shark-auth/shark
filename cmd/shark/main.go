package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
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
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.Storage.Path)
	if dataDir != "" && dataDir != "." {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory %s: %v", dataDir, err)
		}
	}

	// Open SQLite database
	store, err := storage.NewSQLiteStore(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := storage.RunMigrations(store.DB(), migrationsFS, "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations complete")

	// Bootstrap admin API key on first run
	adminCount, err := store.CountActiveAPIKeysByScope(context.Background(), "*")
	if err != nil {
		log.Fatalf("Failed to check admin keys: %v", err)
	}
	if adminCount == 0 {
		fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
		if err != nil {
			log.Fatalf("Failed to generate admin key: %v", err)
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
			log.Fatalf("Failed to create admin key: %v", err)
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
		log.Println("Using Resend HTTP API for email delivery")
		emailSender = email.NewResendSender(cfg.SMTP)
	} else {
		log.Println("Using SMTP for email delivery")
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
		log.Printf("SharkAuth starting on %s", addr)
		log.Printf("Admin dashboard: http://localhost:%d/admin", cfg.Server.Port)
		log.Printf("Health check: http://localhost:%d/healthz", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
