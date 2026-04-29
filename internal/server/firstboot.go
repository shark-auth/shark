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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/mattn/go-isatty"

	"github.com/shark-auth/shark/internal/auth"
	jwtpkg "github.com/shark-auth/shark/internal/auth/jwt"
	"github.com/shark-auth/shark/internal/cli"
	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/storage"
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
	// Migration seeds the row with '{}' â€” treat that as "not configured yet".
	if payload != "" && payload != "{}" {
		return false, nil
	}

	// Check secrets table â€” any row means already bootstrapped.
	_, err = store.GetSecret(ctx, "server.secret")
	if err == nil {
		// Row exists â€” not first boot.
		return false, nil
	}
	if err != sql.ErrNoRows {
		return false, fmt.Errorf("firstboot: check secrets: %w", err)
	}
	return true, nil
}

// PromptBootstrapConfig optionally asks the operator for DB path and port
// before the storage is opened. Only runs on first boot (no state file)
// and TTY.
func PromptBootstrapConfig(cfg *config.Config, opts Options) error {
	if opts.NoPrompt || !isatty.IsTerminal(os.Stdin.Fd()) {
		return nil
	}

	// If ~/.shark/state exists, we've already bootstrapped.
	path, _ := config.SharkStatePath()
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	// ---- Step 3: Interactive prompt (TTY only) ----
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
			if p, err := strconv.Atoi(newPort); err == nil && p > 0 {
				cfg.Server.Port = p
			}
		}
		_ = newBind // bind address is noted for operator reference
	}
	return nil
}

// RunFirstBoot performs all first-boot secret generation and DB seeding.
// It is idempotent: if called on a non-first-boot store it returns nil,nil.
//
// Steps:
//  1. Detect first boot via isFirstBoot
//  2. Generate server.secret, jwt.signing_key.bootstrap, admin.api_key
//  3. Store secrets in DB
//  4. Write ~/.shark/state
//  5. Print generated values
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

	// Branded header moved to server startup or PromptBootstrapConfig.
	// Prompt logic moved to PromptBootstrapConfig.

	// ---- Step 4: Generate secrets ----

	// 4a: server.secret â€” 32-byte hex
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("firstboot: generate server secret: %w", err)
	}
	serverSecret := hex.EncodeToString(secretBytes)

	// Apply to config so JWT manager uses it immediately
	cfg.Server.Secret = serverSecret

	// 4b: JWT signing key via the JWT manager
	// We need a temporary store-backed manager to generate the key.
	// Use empty JWT config â€” EnsureActiveKey will generate an RS256 key.
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
	fmt.Println("  âœ“ Generated server secret")
	fmt.Printf("  âœ“ Generated JWT signing key (kid: bootstrap-%s)\n", kidShort)
	fmt.Println("  âœ“ Generated admin API key")
	// W18: full admin key surfaced via cli.PrintAdminKeyBanner in server.Serve
	// (wide one-time banner). Old masked-print here removed to avoid duplicate.

	return &FirstBootResult{
		AdminKey:   fullKey,
		SetupToken: "", // filled in by server.go after API server is wired
		JWTKid:     "bootstrap-" + kidShort,
	}, nil
}

// RunRerunBanner prints an operator-oriented status banner on second+ boots.
// It is called AFTER RunFirstBoot (which handles true first-boot) and AFTER
// the HTTP server reports "listening", so the operator always knows the state
// of their installation without reading log lines.
//
// Three states:
//  1. Active admin API key exists â†’ "âœ“ admin configured" + dashboard URL
//  2. No admin key + bootstrap key file present â†’ "âš  setup pending" + key file path
//  3. No admin key + no key file â†’ nothing (first-boot flow already ran above)
func RunRerunBanner(ctx context.Context, store *storage.SQLiteStore, cfg *config.Config) {
	// Determine dashboard URL from configured base_url, falling back to port.
	dashURL := cfg.Server.BaseURL
	if dashURL == "" {
		port := cfg.Server.Port
		if port == 0 {
			port = 8080
		}
		dashURL = fmt.Sprintf("http://localhost:%d", port)
	}
	dashURL = strings.TrimRight(dashURL, "/") + "/admin"

	// Check whether any active admin-scoped API key exists.
	count, err := store.CountActiveAPIKeysByScope(ctx, "*")
	if err != nil {
		slog.Warn("rerun-banner: could not query api_keys", "err", err)
		return
	}

	if count > 0 {
		// State 1 â€” admin already configured.
		cli.PrintAdminConfigured(dashURL)
		return
	}

	// State 2 â€” no admin key yet, check for bootstrap key file.
	keyPath := filepath.Join(filepath.Dir(cfg.Storage.Path), "admin.key.firstboot")
	if _, statErr := os.Stat(keyPath); statErr == nil {
		cli.PrintSetupPending(dashURL, keyPath)
		return
	}

	// State 3 â€” no key, no file: first-boot flow already handled this boot.
	// Nothing to print here; the first-boot banner already ran.
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
