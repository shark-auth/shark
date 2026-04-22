package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// base62Alphabet is the standard base62 character set (0-9, A-Z, a-z).
const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// base62Encode encodes a byte slice to a base62 string by treating the bytes as
// a big-endian unsigned integer and repeatedly dividing by 62.
func base62Encode(b []byte) string {
	// Work with a mutable copy of the bytes (big-endian big integer).
	num := make([]byte, len(b))
	copy(num, b)

	var result []byte
	for !isZero(num) {
		rem := divmod(num, 62)
		result = append(result, base62Alphabet[rem])
	}
	// Result is in reverse order.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	if len(result) == 0 {
		return "0"
	}
	return string(result)
}

// isZero returns true if all bytes are zero.
func isZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// divmod divides the big-endian byte slice n by d in place and returns the remainder.
func divmod(n []byte, d byte) byte {
	var rem uint64
	for i := range n {
		cur := rem*256 + uint64(n[i])
		n[i] = byte(cur / uint64(d)) //#nosec G115 -- base-62 long division: d is a byte (≤255) and cur/d fits in a byte by construction
		rem = cur % uint64(d)
	}
	return byte(rem)
}

// generateClientSecret produces a 32-byte random secret encoded in base62 (~43 chars).
func generateClientSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base62Encode(b), nil
}

// seedDefaultApplication creates the default OAuth application on first boot.
// It is idempotent: if a default application already exists, it returns nil.
func seedDefaultApplication(ctx context.Context, store storage.Store, cfg *config.Config) error {
	_, err := store.GetDefaultApplication(ctx)
	if err == nil {
		// Already exists — idempotent, no-op.
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// Generate client ID.
	nid, err := gonanoid.New(21)
	if err != nil {
		return fmt.Errorf("generate client id: %w", err)
	}
	clientID := "shark_app_" + nid

	// Generate client secret (32 bytes, base62-encoded).
	clientSecret, err := generateClientSecret()
	if err != nil {
		return fmt.Errorf("generate client secret: %w", err)
	}

	// Hash the secret.
	h := sha256.Sum256([]byte(clientSecret))
	secretHash := hex.EncodeToString(h[:])
	secretPrefix := clientSecret
	if len(secretPrefix) > 8 {
		secretPrefix = secretPrefix[:8]
	}

	// Build allowed callback URLs from config.
	var callbacks []string
	if cfg.Social.RedirectURL != "" {
		callbacks = append(callbacks, cfg.Social.RedirectURL)
	}
	if cfg.MagicLink.RedirectURL != "" {
		distinct := true
		for _, u := range callbacks {
			if u == cfg.MagicLink.RedirectURL {
				distinct = false
				break
			}
		}
		if distinct {
			callbacks = append(callbacks, cfg.MagicLink.RedirectURL)
		}
	}
	if callbacks == nil {
		callbacks = []string{}
	}

	// Build the application ID.
	appNid, err := gonanoid.New()
	if err != nil {
		return fmt.Errorf("generate app id: %w", err)
	}
	now := time.Now().UTC()
	app := &storage.Application{
		ID:                  "app_" + appNid,
		Name:                "Default Application",
		Slug:                "default",
		ClientID:            clientID,
		ClientSecretHash:    secretHash,
		ClientSecretPrefix:  secretPrefix,
		AllowedCallbackURLs: callbacks,
		AllowedLogoutURLs:   []string{},
		AllowedOrigins:      []string{},
		IsDefault:           true,
		Metadata:            map[string]any{},
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := store.CreateApplication(ctx, app); err != nil {
		// Handle concurrent first-boot race: another process already inserted.
		// Confirm by re-fetching.
		if _, rerr := store.GetDefaultApplication(ctx); rerr == nil {
			return nil
		}
		return fmt.Errorf("create default application: %w", err)
	}

	printDefaultAppBanner(clientID, clientSecret)
	return nil
}

func printDefaultAppBanner(clientID, clientSecret string) {
	fmt.Println()
	fmt.Println("  ============================================================")
	fmt.Println("    Default application created")
	fmt.Printf("    client_id:     %s\n", clientID)
	fmt.Printf("    client_secret: %s   (shown once — save it)\n", clientSecret)
	fmt.Println("  ============================================================")
	fmt.Println()
}
