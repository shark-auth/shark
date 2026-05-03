package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/shark-auth/shark/internal/storage"
)

// Bootstrap token (T15) â€” one-time URL that mints a short-lived admin API key
// so a fresh install doesn't force the operator to paste sk_live_â€¦ from the
// server log. Flow:
//   1. On `shark serve` startup, if no admin audit events exist, the caller
//      (server.Serve) invokes Server.MintBootstrapToken and prints the URL to
//      stdout.
//   2. The dashboard's login.tsx reads ?bootstrap=<tok> and POSTs here.
//   3. We verify the token (hash-compared, not-expired, not-consumed), mint a
//      real admin API key with scopes ["*"], mark the token consumed, and
//      return {api_key} to the browser. The browser drops the key into
//      sessionStorage and reloads the dashboard.
//
// Security:
//   - crypto/rand, 32 bytes â†’ base16 (64 chars)
//   - only the SHA-256 hash is kept in memory; the raw token exists for one
//     stdout line and one HTTP request
//   - single-use (consumed=true flips on first match)
//   - 10-minute expiry
//   - in-memory only: restarting the server invalidates all outstanding
//     tokens, which matches the "first start" intent
//   - no auth middleware (this IS the auth bootstrap), but the consume
//     endpoint 401s for any input that doesn't match the stored hash

type bootstrapToken struct {
	hash      string // hex(sha256(raw))
	expiresAt time.Time
	consumed  bool
}

var (
	bootstrapMu     sync.Mutex
	bootstrapToken_ *bootstrapToken //nolint:revive // package-private singleton
)

// bootstrapTokenTTL is the window in which the printed URL is valid. Kept
// short because the token grants full admin â€” if the operator doesn't click
// within 10 minutes, they can restart the server to get a fresh URL.
const bootstrapTokenTTL = 10 * time.Minute

// MintBootstrapToken generates a one-time token, stores its hash in-memory,
// and returns the raw token for stdout. Calling it again replaces any prior
// token (so repeated startups don't leave stale tokens lying around).
//
// Returns ("", nil) when an admin has already been bootstrapped â€” detected
// via presence of any audit_logs row with action LIKE 'admin.%'. The caller
// uses "" to mean "don't print anything".
func (s *Server) MintBootstrapToken(ctx context.Context) (string, error) {
	// Bootstrap is complete when the firstboot key file no longer exists.
	keyPath := s.firstbootKeyPath()
	if keyPath != "" {
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			return "", nil
		}
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	token := hex.EncodeToString(raw)
	h := sha256.Sum256([]byte(token))

	bootstrapMu.Lock()
	bootstrapToken_ = &bootstrapToken{
		hash:      hex.EncodeToString(h[:]),
		expiresAt: time.Now().UTC().Add(bootstrapTokenTTL),
		consumed:  false,
	}
	bootstrapMu.Unlock()

	return token, nil
}

// bootstrapConsumeRequest is the JSON body for POST /admin/bootstrap/consume.
type bootstrapConsumeRequest struct {
	Token string `json:"token"`
}

// bootstrapConsumeResponse returns the fresh admin API key on success.
type bootstrapConsumeResponse struct {
	APIKey string `json:"api_key"`
}

// handleBootstrapConsume validates a bootstrap token and mints a real admin
// API key. No auth middleware is mounted on this route â€” the token itself is
// the credential.
func (s *Server) handleBootstrapConsume(w http.ResponseWriter, r *http.Request) {
	var req bootstrapConsumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "token is required",
		})
		return
	}

	suppliedHash := sha256.Sum256([]byte(req.Token))
	suppliedHex := hex.EncodeToString(suppliedHash[:])

	bootstrapMu.Lock()
	tok := bootstrapToken_
	// Compare + flip consumed under the same lock so two racing requests
	// can't both win. On mismatch/expired/consumed we return 401.
	if tok == nil || tok.consumed || time.Now().After(tok.expiresAt) {
		bootstrapMu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_token",
			"message": "Bootstrap token is invalid, expired, or already used.",
		})
		return
	}
	if subtle.ConstantTimeCompare([]byte(tok.hash), []byte(suppliedHex)) != 1 {
		bootstrapMu.Unlock()
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "invalid_token",
			"message": "Bootstrap token is invalid, expired, or already used.",
		})
		return
	}
	// Mark consumed BEFORE returning the key so a partial failure below
	// doesn't leave the token replayable.
	tok.consumed = true
	bootstrapMu.Unlock()

	keyPath := s.firstbootKeyPath()
	if keyPath == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "First-boot key path not configured.",
		})
		return
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		writeJSON(w, http.StatusGone, map[string]string{
			"error":   "not_found",
			"message": "First-boot key file not found. It may have already been consumed or deleted.",
		})
		return
	}

	fullKey := strings.TrimSpace(string(data))

	// Delete the file from disk so it's no longer sitting around in plaintext.
	_ = os.Remove(keyPath)

	// Attempt to find the default-admin key in the DB for the audit log
	keys, _ := s.Store.ListAPIKeys(r.Context())
	var defaultAdminKey *storage.APIKey
	for _, k := range keys {
		if k.Name == "default-admin" {
			defaultAdminKey = k
			break
		}
	}

	// Audit trail: future `shark serve` runs will see this admin.* event and
	// skip the token print.
	if s.AuditLogger != nil {
		now := time.Now().UTC().Format(time.RFC3339)
		id, _ := gonanoid.New()

		actorID := "admin_key"
		prefix := "unknown"
		if defaultAdminKey != nil {
			actorID = defaultAdminKey.ID
			prefix = defaultAdminKey.KeyPrefix
			if prefix == "" && len(defaultAdminKey.ID) >= 12 {
				prefix = defaultAdminKey.ID[:12]
			}
		}

		metaBytes, _ := json.Marshal(map[string]any{
			"key_prefix":  prefix,
			"consumed_at": now,
			"action":      "key_file_deleted",
		})
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ID:         "audit_" + id,
			ActorID:    actorID,
			ActorType:  "admin",
			Action:     "admin.bootstrap.consumed",
			TargetType: "api_key",
			TargetID:   actorID,
			IP:         ipOf(r),
			UserAgent:  uaOf(r),
			Metadata:   string(metaBytes),
			Status:     "success",
			CreatedAt:  now,
		})
	}

	writeJSON(w, http.StatusOK, bootstrapConsumeResponse{APIKey: fullKey})
}

// handleFirstbootKey always returns 410 Gone and never exposes the firstboot
// key. The key is only available via the one-time bootstrap flow.
//
// GET /api/v1/admin/firstboot/key
func (s *Server) handleFirstbootKey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, map[string]string{
		"error":   "gone",
		"message": "First-boot key has been consumed or is unavailable.",
	})
}

// firstbootKeyPath returns the expected path of the first-boot key file based
// on the server's configured storage path. Returns "" when config is missing.
func (s *Server) firstbootKeyPath() string {
	if s.Config == nil {
		return ""
	}
	p := s.Config.Storage.Path
	if p == "" {
		p = "./shark.db"
	}
	dir := filepath.Dir(p)
	return filepath.Join(dir, "admin.key.firstboot")
}
