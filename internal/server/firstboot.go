package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/mattn/go-isatty"

	"github.com/sharkauth/sharkauth/internal/auth"
	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// FirstBootResult holds secrets generated on first boot that must be shown
// once to the operator and then discarded.
type FirstBootResult struct {
	// AdminKey is the full sk_live_* admin API key (shown once, hash stored in DB).
	AdminKey string
	// SetupToken is the one-time setup URL token stored in the API server.
	SetupToken string
	// JWTKid is the key ID of the generated JWT signing key (for display).
	JWTKid string
}

// isFirstBoot returns true when both system_config payload is "{}" (or empty)
// AND the secrets table has no rows.
func isFirstBoot(ctx context.Context, store *storage.SQLiteStore) (bool, error) {
	payload, err := store.GetSystemConfig(ctx)
	if err != nil {
		return false, fmt.Errorf("firstboot: get system_config: %w", err)
	}
	// Migration seeds the row with '{}' — treat that as "not configured yet".
	if payload != "" && payload != "{}" {
		return false, nil
	}

	// Check secrets table — any row means already bootstrapped.
	_, err = store.GetSecret(ctx, "server.secret")
	if err == nil {
		// Row exists — not first boot.
		return false, nil
	}
	if err != sql.ErrNoRows {
		return false, fmt.Errorf("firstboot: check secrets: %w", err)
	}
	return true, nil
}

// RunFirstBoot performs all first-boot secret generation and DB seeding.
// It is idempotent: if called on a non-first-boot store it returns nil,nil.
//
// Steps:
//  1. Detect first boot via isFirstBoot
//  2. Print branded ASCII header
//  3. Prompt for db_path/port/bind if TTY (unless NoPrompt)
//  4. Generate server.secret, jwt.signing_key.bootstrap, admin.api_key
//  5. Store secrets in DB
//  6. Write ~/.shark/state
//  7. Print generated values
func RunFirstBoot(ctx context.Context, store *storage.SQLiteStore, cfg *config.Config, opts Options) (*FirstBootResult, error) {
	first, err := isFirstBoot(ctx, store)
	if err != nil {
		return nil, err
	}
	if !first {
		// Non-first-boot: load existing server.secret from DB into cfg so the
		// caller's secret-length validation (server.go) passes.
		existingSecret, err := store.GetSecret(ctx, "server.secret")
		if err != nil {
			return nil, fmt.Errorf("firstboot: load server.secret on subsequent boot: %w", err)
		}
		cfg.Server.Secret = existingSecret
		return nil, nil
	}

	// ---- Step 2: Branded header ----
	fmt.Println()
	fmt.Println("==== SharkAuth — Open Source Auth for Agents and Humans ====")
	fmt.Println()

	// ---- Step 3: Interactive prompt (TTY only) ----
	if !opts.NoPrompt && isatty.IsTerminal(os.Stdin.Fd()) {
		dbPath := cfg.Storage.Path
		if dbPath == "" {
			dbPath = "./shark.db"
		}
		port := fmt.Sprintf("%d", cfg.Server.Port)
		if cfg.Server.Port == 0 {
			port = "8080"
		}
		bind := "0.0.0.0"

		fmt.Printf("  Use defaults? (db=%s, port=%s, bind=%s) [Y/n]: ", dbPath, port, bind)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))

		if answer == "n" || answer == "no" {
			newDB := prompt(reader, "  db path", dbPath)
			newPort := prompt(reader, "  port", port)
			newBind := prompt(reader, "  bind address", bind)
			if newDB != "" && newDB != dbPath {
				cfg.Storage.Path = newDB
			}
			if newPort != "" && newPort != port {
				// Note: port is used for display; the actual listen port comes from cfg
				_ = newPort // stored in cfg.Server.Port by serve.go flags
			}
			_ = newBind // bind address is set via flags; noted for operator reference
		}
	} else {
		slog.Info("first boot: non-TTY detected, using defaults silently")
	}

	// ---- Step 4: Generate secrets ----

	// 4a: server.secret — 32-byte hex
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("firstboot: generate server secret: %w", err)
	}
	serverSecret := hex.EncodeToString(secretBytes)

	// Apply to config so JWT manager uses it immediately
	cfg.Server.Secret = serverSecret

	// 4b: JWT signing key via the JWT manager
	// We need a temporary store-backed manager to generate the key.
	// Use empty JWT config — EnsureActiveKey will generate an RS256 key.
	jwtMgr := jwtpkg.NewManager(&cfg.Auth.JWT, store, cfg.Server.BaseURL, serverSecret)
	if err := jwtMgr.GenerateAndStore(ctx, false); err != nil {
		return nil, fmt.Errorf("firstboot: generate JWT signing key: %w", err)
	}
	activeKey, err := store.GetActiveSigningKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("firstboot: get active signing key: %w", err)
	}
	jwtKid := activeKey.KID

	// 4c: admin API key
	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("firstboot: generate admin API key: %w", err)
	}

	// ---- Step 5: Store secrets in DB ----
	storeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := store.SetSecret(storeCtx, "server.secret", serverSecret); err != nil {
		return nil, fmt.Errorf("firstboot: store server.secret: %w", err)
	}
	if err := store.SetSecret(storeCtx, "admin.api_key.hash", keyHash); err != nil {
		return nil, fmt.Errorf("firstboot: store admin.api_key.hash: %w", err)
	}
	if err := store.SetSecret(storeCtx, "admin.api_key.prefix", keyPrefix); err != nil {
		return nil, fmt.Errorf("firstboot: store admin.api_key.prefix: %w", err)
	}
	if err := store.SetSecret(storeCtx, "admin.api_key.suffix", keySuffix); err != nil {
		return nil, fmt.Errorf("firstboot: store admin.api_key.suffix: %w", err)
	}

	// Also create the admin API key in the api_keys table so the existing
	// admin middleware (AdminAPIKeyFromStore) can verify it immediately.
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
	if err := store.CreateAPIKey(storeCtx, k); err != nil {
		return nil, fmt.Errorf("firstboot: create api key record: %w", err)
	}

	// ---- Step 6: Write ~/.shark/state ----
	if err := config.WriteModeState("prod"); err != nil {
		slog.Warn("firstboot: could not write ~/.shark/state", "err", err)
		// Non-fatal
	}

	// ---- Step 7: Print confirmation ----
	kidShort := jwtKid
	if len(kidShort) > 8 {
		kidShort = kidShort[:8]
	}
	maskedKey := fullKey
	if len(maskedKey) > 20 {
		maskedKey = maskedKey[:12] + "..." + maskedKey[len(maskedKey)-4:]
	}

	fmt.Println("  ✓ Generated server secret")
	fmt.Printf("  ✓ Generated JWT signing key (kid: bootstrap-%s)\n", kidShort)
	fmt.Println("  ✓ Generated admin API key")
	fmt.Println()
	fmt.Printf("  Admin key: %s  (save this — you can't see it again)\n", maskedKey)
	fmt.Println()

	return &FirstBootResult{
		AdminKey:   fullKey,
		SetupToken: "", // filled in by server.go after API server is wired
		JWTKid:     "bootstrap-" + kidShort,
	}, nil
}

// prompt prints a prompt with a default and reads a line from r.
// Returns the default when the user presses Enter without input.
func prompt(r *bufio.Reader, label, defaultVal string) string {
	fmt.Printf("  %s [%s]: ", label, defaultVal)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}
