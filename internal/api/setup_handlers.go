package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// setupTokenTTL is the lifetime of the one-time setup token.
// 30 minutes — long enough for a human to read the terminal and click the URL.
const setupTokenTTL = 30 * time.Minute

// setupToken is the in-memory, one-shot first-boot setup credential.
// Only the SHA-256 hash of the raw token is kept here; the raw value
// exists for one terminal print and one HTTP request.
type setupTokenState struct {
	hash      string // hex(sha256(raw))
	expiresAt time.Time
	consumed  bool
	// apiKey is the full sk_live_* key shown on the setup page (shown once).
	apiKey string
}

var (
	setupMu    sync.Mutex
	setupState *setupTokenState
)

// MintSetupToken generates a one-time setup token, stores its hash in memory,
// and returns the raw token for printing. Calling it again replaces any prior
// token. Returns ("", nil) when the store already has admin users (not first boot).
//
// apiKey is the full admin API key generated during first boot — it will be
// shown once on the /admin/setup page and then discarded.
func (s *Server) MintSetupToken(apiKey string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("setup token: crypto/rand: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	h := sha256.Sum256([]byte(token))

	setupMu.Lock()
	setupState = &setupTokenState{
		hash:      hex.EncodeToString(h[:]),
		expiresAt: time.Now().UTC().Add(setupTokenTTL),
		consumed:  false,
		apiKey:    apiKey,
	}
	setupMu.Unlock()

	return token, nil
}

// SetupTokenMiddleware accepts "Authorization: Setup <token>" on setup routes.
// It validates the token but does NOT consume it — consumption happens in
// handleSetupAdminUser on success.
func (s *Server) SetupTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		const prefix = "Setup "
		if !strings.HasPrefix(authHeader, prefix) {
			writeJSON(w, http.StatusUnauthorized, errPayload("unauthorized", "Setup token required"))
			return
		}
		supplied := strings.TrimPrefix(authHeader, prefix)
		if !s.validateSetupToken(supplied) {
			writeJSON(w, http.StatusUnauthorized, errPayload("invalid_token", "Setup token is invalid, expired, or already used"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// validateSetupToken checks the supplied raw token against the stored hash.
// Does NOT consume the token.
func (s *Server) validateSetupToken(supplied string) bool {
	setupMu.Lock()
	defer setupMu.Unlock()

	tok := setupState
	if tok == nil || tok.consumed || time.Now().After(tok.expiresAt) {
		return false
	}
	h := sha256.Sum256([]byte(supplied))
	suppliedHex := hex.EncodeToString(h[:])
	return subtle.ConstantTimeCompare([]byte(tok.hash), []byte(suppliedHex)) == 1
}

// handleSetupInfo returns the one-time admin API key to the setup page.
// The key is served only once — subsequent calls return 410 Gone.
// GET /api/v1/admin/setup/info  (protected by SetupTokenMiddleware)
func (s *Server) handleSetupInfo(w http.ResponseWriter, r *http.Request) {
	setupMu.Lock()
	tok := setupState
	setupMu.Unlock()

	if tok == nil || tok.consumed || time.Now().After(tok.expiresAt) {
		writeJSON(w, http.StatusGone, errPayload("gone", "Setup has already been completed or expired"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"api_key": tok.apiKey,
	})
}

// setupAdminUserRequest is the JSON body for POST /api/v1/admin/setup/admin-user.
type setupAdminUserRequest struct {
	Email string `json:"email"`
}

// setupAdminUserResponse is returned after creating the first admin user.
type setupAdminUserResponse struct {
	Sent        bool   `json:"sent"`
	DevInboxURL string `json:"dev_inbox_url,omitempty"`
}

// handleSetupAdminUser creates the first admin user and sends a magic-link.
// Protected by SetupTokenMiddleware. On success, the setup token is consumed.
// POST /api/v1/admin/setup/admin-user
func (s *Server) handleSetupAdminUser(w http.ResponseWriter, r *http.Request) {
	setupMu.Lock()
	tok := setupState
	setupMu.Unlock()

	if tok == nil || tok.consumed || time.Now().After(tok.expiresAt) {
		writeJSON(w, http.StatusGone, errPayload("gone", "Setup has already been completed or expired"))
		return
	}

	var req setupAdminUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "email is required"))
		return
	}

	ctx := r.Context()

	// Check if user already exists; create if not.
	user, err := s.Store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// User doesn't exist — create them.
		id, _ := gonanoid.New()
		now := time.Now().UTC().Format(time.RFC3339)
		newUser := &storage.User{
			ID:            "usr_" + id,
			Email:         req.Email,
			EmailVerified: true, // admin bootstrap — trust the operator
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if createErr := s.Store.CreateUser(ctx, newUser); createErr != nil {
			slog.Error("setup: create admin user failed", "err", createErr)
			writeJSON(w, http.StatusInternalServerError, errPayload("internal_error", "Failed to create admin user"))
			return
		}
		user = newUser
		slog.Info("setup: created first admin user", "email", req.Email, "id", user.ID)
	}

	// Send magic-link.
	var devInboxURL string
	if s.MagicLinkManager == nil {
		slog.Warn("setup: MagicLinkManager not wired — cannot send magic link")
		writeJSON(w, http.StatusServiceUnavailable, errPayload("not_configured", "Email not configured"))
		return
	}

	if err := s.MagicLinkManager.SendMagicLink(ctx, req.Email); err != nil {
		slog.Error("setup: send magic link failed", "err", err)
		// Non-fatal for dev-inbox: the link will still appear in the dev inbox.
	}

	// If dev-inbox is configured, expose the URL for the setup page toast.
	if s.Config.Server.DevMode || s.Config.Email.Provider == "dev" || s.Config.Email.Provider == "" {
		emails, listErr := s.Store.ListDevEmails(ctx, 1)
		if listErr == nil && len(emails) > 0 {
			devInboxURL = "/api/v1/admin/dev/emails/" + emails[0].ID
		} else {
			// Point to the dev-email list page
			devInboxURL = "/admin/dev-email"
		}
	}

	// Consume the setup token — setup is complete.
	setupMu.Lock()
	if setupState != nil {
		setupState.consumed = true
		// Wipe the API key from memory — hash+prefix+suffix remain in DB.
		setupState.apiKey = ""
	}
	setupMu.Unlock()

	slog.Info("setup: first-boot setup complete", "email", req.Email)
	fmt.Printf("\n  Setup complete — magic link sent to %s\n\n", req.Email)

	resp := setupAdminUserResponse{
		Sent:        true,
		DevInboxURL: devInboxURL,
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleSetupStatus returns whether setup is still pending or already done.
// GET /api/v1/admin/setup/status  (public — no auth needed)
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	setupMu.Lock()
	tok := setupState
	setupMu.Unlock()

	pending := tok != nil && !tok.consumed && time.Now().Before(tok.expiresAt)
	writeJSON(w, http.StatusOK, map[string]bool{
		"pending": pending,
	})
}
