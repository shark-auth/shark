# SharkAuth Security Audit & Bug Fixes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the silent email failure, patch all security vulnerabilities found during audit, and repair broken features (OAuth, env var config) that appear functional but silently fail.

**Architecture:** 13 targeted fixes across config loading, middleware, session management, OAuth wiring, and email error handling. Each task is a self-contained fix with its own commit. No structural refactors â€” surgical changes only.

**Tech Stack:** Go 1.25, Chi v5, gorilla/securecookie, koanf v2, SQLite, crypto/sha256, crypto/subtle

---

## Task 1: Fix env var interpolation in YAML config

The root cause of the email bug. Koanf does NOT resolve `${VAR_NAME}` syntax in YAML values â€” the literal string `"${RESEND_API_KEY}"` is used as the SMTP password. This also breaks `server.secret`, `admin.api_key`, and all OAuth credentials.

**Files:**
- Modify: `internal/config/config.go:210-214`

- [ ] **Step 1: Add env var interpolation after YAML load**

In `internal/config/config.go`, add an `os` import and a helper function, then call it after YAML loading:

```go
import (
	"os"
	"regexp"
	// ... existing imports
)

// envVarPattern matches ${VAR_NAME} patterns in config values.
var envVarPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// interpolateEnvVars walks all koanf keys and replaces ${VAR} patterns
// with actual environment variable values.
func interpolateEnvVars(k *koanf.Koanf) {
	for _, key := range k.Keys() {
		val := k.String(key)
		if val == "" {
			continue
		}
		replaced := envVarPattern.ReplaceAllStringFunc(val, func(match string) string {
			varName := envVarPattern.FindStringSubmatch(match)[1]
			if envVal, ok := os.LookupEnv(varName); ok {
				return envVal
			}
			return match // leave unresolved if env var not set
		})
		if replaced != val {
			k.Set(key, replaced)
		}
	}
}
```

Then in the `Load` function, after the YAML load and before the env.Provider load, add:

```go
	// Load YAML file if it exists
	if path != "" {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file %s: %w", path, err)
		}
	}

	// Interpolate ${VAR_NAME} patterns in YAML values with actual env vars
	interpolateEnvVars(k)

	// Load environment variable overrides with SHARKAUTH_ prefix
```

- [ ] **Step 2: Run the build to verify compilation**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "fix: interpolate \${VAR} env vars in YAML config values

Koanf does not natively resolve \${VAR_NAME} syntax in YAML.
The SMTP password, server secret, admin key, and OAuth credentials
were all being used as literal strings like '\${RESEND_API_KEY}'.
This is the root cause of the silent email failure."
```

---

## Task 2: Fix env var override mapping for underscored config keys

`SHARKAUTH_SERVER_BASE_URL` maps to `server.base.url` instead of `server.base_url` because the transformer replaces ALL underscores with dots. This breaks env var overrides for `base_url`, `session_lifetime`, `password_min_length`, `from_name`, `token_lifetime`, `redirect_url`, `client_id`, `client_secret`, `api_key`, etc.

**Files:**
- Modify: `internal/config/config.go:217-224`

- [ ] **Step 1: Replace naive underscore mapping with depth-aware mapper**

The koanf structure has known nesting depths. The fix: use double-underscore `__` as the nesting separator, single underscore is literal. This is the standard convention (used by Spring Boot, ASP.NET, etc.).

Replace the env.Provider block in `Load()`:

```go
	// Load environment variable overrides with SHARKAUTH_ prefix.
	// Nesting uses double-underscore: SHARKAUTH_SMTP__FROM_NAME -> smtp.from_name
	// Single underscores are preserved as literal underscores in key names.
	if err := k.Load(env.Provider("SHARKAUTH_", ".", func(s string) string {
		key := strings.TrimPrefix(s, "SHARKAUTH_")
		key = strings.ToLower(key)
		// Double underscore is the nesting separator
		key = strings.ReplaceAll(key, "__", ".")
		return key
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}
```

- [ ] **Step 2: Run the build to verify compilation**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "fix: use double-underscore as env var nesting separator

SHARKAUTH_SERVER_BASE_URL mapped to server.base.url (wrong).
Now uses __ for nesting: SHARKAUTH_SMTP__FROM_NAME -> smtp.from_name.
Single underscores are preserved as literal underscores in key names.
This matches the convention used by Spring Boot and ASP.NET Core."
```

---

## Task 3: Log email send errors instead of silently swallowing them

The magic link handler discards the send error with `_ =`, so SMTP failures are invisible. The handler should still return 200 (to prevent email enumeration), but must log the error.

**Files:**
- Modify: `internal/api/magiclink_handlers.go:82-85`

- [ ] **Step 1: Add error logging**

Add a `"log"` import to the file, then replace the send block:

```go
	// Send magic link (always return success to avoid leaking info about email existence)
	if s.MagicLinkManager != nil {
		if err := s.MagicLinkManager.SendMagicLink(r.Context(), req.Email); err != nil {
			log.Printf("ERROR: failed to send magic link email to %s: %v", req.Email, err)
		}
	}
```

- [ ] **Step 2: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/api/magiclink_handlers.go
git commit -m "fix: log email send errors instead of silently discarding

Magic link handler was using _ = to discard SMTP errors.
Now logs the error while still returning 200 to prevent
email enumeration. This makes SMTP failures visible in logs."
```

---

## Task 4: Fix unauthenticated `/auth/check` endpoint (IDOR)

`POST /api/v1/auth/check` is mounted on a public route with no auth middleware. Anyone can query any user's permissions. Move it behind `AdminAPIKey` middleware.

**Files:**
- Modify: `internal/api/router.go:111`

- [ ] **Step 1: Move `/auth/check` behind admin API key middleware**

In `router.go`, remove line 111 (`r.Post("/check", s.handleAuthCheck)`) from the public auth group and add it to a new admin-protected group. Replace this section:

```go
			r.Post("/check", s.handleAuthCheck)
```

With nothing (delete the line). Then add a new admin-protected route block right after the existing `/permissions` route block (after line 185):

```go
		// Auth check (admin) â€” validates if a user has a specific permission
		r.Group(func(r chi.Router) {
			r.Use(mw.AdminAPIKey(cfg.Admin.APIKey))
			r.Post("/auth/check", s.handleAuthCheck)
		})
```

- [ ] **Step 2: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/api/router.go
git commit -m "fix(security): require admin API key for /auth/check endpoint

The permission check endpoint was publicly accessible, allowing
anyone to enumerate permissions for any user_id (IDOR vulnerability).
Now requires X-Admin-Key header like all other admin endpoints."
```

---

## Task 5: Register OAuth providers in OAuthManager

`initOAuthManager()` creates an empty OAuthManager but never calls `RegisterProvider()`. All OAuth login attempts fail with "unsupported provider". The provider implementations exist in `internal/auth/providers/` but are never wired up.

**Files:**
- Modify: `internal/api/oauth_handlers.go:130-133`

- [ ] **Step 1: Wire up all configured OAuth providers**

Add the providers import and replace `initOAuthManager`:

```go
import (
	// ... existing imports
	"github.com/shark-auth/shark/internal/auth/providers"
)
```

Replace the `initOAuthManager` function:

```go
// initOAuthManager creates the OAuthManager and registers configured providers.
func (s *Server) initOAuthManager() {
	s.OAuthManager = auth.NewOAuthManager(s.Store, s.SessionManager, s.Config)

	baseURL := s.Config.Server.BaseURL

	// Register Google if configured
	if s.Config.Social.Google.ClientID != "" && s.Config.Social.Google.ClientSecret != "" {
		s.OAuthManager.RegisterProvider(providers.NewGoogle(s.Config.Social.Google, baseURL))
	}

	// Register GitHub if configured
	if s.Config.Social.GitHub.ClientID != "" && s.Config.Social.GitHub.ClientSecret != "" {
		s.OAuthManager.RegisterProvider(providers.NewGitHub(s.Config.Social.GitHub, baseURL))
	}

	// Register Apple if configured
	if s.Config.Social.Apple.ClientID != "" && s.Config.Social.Apple.TeamID != "" {
		s.OAuthManager.RegisterProvider(providers.NewApple(s.Config.Social.Apple, baseURL))
	}

	// Register Discord if configured
	if s.Config.Social.Discord.ClientID != "" && s.Config.Social.Discord.ClientSecret != "" {
		s.OAuthManager.RegisterProvider(providers.NewDiscord(s.Config.Social.Discord, baseURL))
	}
}
```

- [ ] **Step 2: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/api/oauth_handlers.go
git commit -m "fix: register OAuth providers (Google, GitHub, Apple, Discord)

initOAuthManager created an empty manager but never called
RegisterProvider. All OAuth login attempts returned 'unsupported
provider'. Now conditionally registers each provider when its
client_id and client_secret are configured."
```

---

## Task 6: Add `Secure` flag to session cookies

Session cookies are never set with `Secure: true`, making them transmittable over plain HTTP. The flag should be set when the configured `base_url` uses HTTPS.

**Files:**
- Modify: `internal/auth/session.go:37,130-145`

- [ ] **Step 1: Add secure flag based on base URL**

First, update the `SessionManager` struct and constructor to store the secure flag. Add a `"strings"` import.

Add a `secure` field to the struct:

```go
type SessionManager struct {
	store       storage.Store
	cookieCodec *securecookie.SecureCookie
	lifetime    time.Duration
	secure      bool
}
```

Update `NewSessionManager` to accept a `baseURL` parameter:

```go
func NewSessionManager(store storage.Store, secret string, lifetime time.Duration, baseURL string) *SessionManager {
	secretBytes := []byte(secret)
	hashKey := secretBytes
	if len(hashKey) > 32 {
		hashKey = hashKey[:32]
	}
	blockKey := make([]byte, 32)
	for i := 0; i < 32; i++ {
		blockKey[i] = secretBytes[i%len(secretBytes)]
	}

	codec := securecookie.New(hashKey, blockKey)

	return &SessionManager{
		store:       store,
		cookieCodec: codec,
		lifetime:    lifetime,
		secure:      strings.HasPrefix(baseURL, "https://"),
	}
}
```

Then in `SetSessionCookie`, add the `Secure` flag:

```go
func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	encoded, err := sm.cookieCodec.Encode(cookieName, sessionID)
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sm.lifetime.Seconds()),
	})
}
```

- [ ] **Step 2: Update the caller in router.go to pass baseURL**

In `internal/api/router.go:50`, update the `NewSessionManager` call:

```go
	sm := auth.NewSessionManager(store, cfg.Server.Secret, sessionLifetime, cfg.Server.BaseURL)
```

- [ ] **Step 3: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/auth/session.go internal/api/router.go
git commit -m "fix(security): set Secure flag on session cookies for HTTPS

Session cookies were never marked Secure, allowing transmission
over plain HTTP and MITM interception. Now automatically sets
Secure: true when base_url uses https://."
```

---

## Task 7: Fix weak block key derivation for session encryption

The current block key derivation is a repeating byte pattern (`secret[i % len]`), not a proper KDF. A 10-char secret yields only 80 bits of entropy repeating 3.2x. Use SHA-256 to derive a proper 32-byte block key.

**Files:**
- Modify: `internal/auth/session.go:38-58`

- [ ] **Step 1: Replace repeating-byte derivation with SHA-256**

Add `"crypto/sha256"` to the imports in `session.go`. Replace the block key derivation in `NewSessionManager`:

```go
func NewSessionManager(store storage.Store, secret string, lifetime time.Duration, baseURL string) *SessionManager {
	secretBytes := []byte(secret)
	// Hash key: use first 32 bytes of secret for HMAC signing
	hashKey := secretBytes
	if len(hashKey) > 32 {
		hashKey = hashKey[:32]
	}
	// Block key: derive a 32-byte AES key using SHA-256
	blockKeyHash := sha256.Sum256(append([]byte("sharkauth-block-key:"), secretBytes...))
	blockKey := blockKeyHash[:]

	codec := securecookie.New(hashKey, blockKey)

	return &SessionManager{
		store:       store,
		cookieCodec: codec,
		lifetime:    lifetime,
		secure:      strings.HasPrefix(baseURL, "https://"),
	}
}
```

Note: this changes the block key derivation, which will invalidate all existing session cookies. Users will need to log in again after this deploy â€” which is acceptable for a pre-alpha security fix.

- [ ] **Step 2: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/auth/session.go
git commit -m "fix(security): derive session block key with SHA-256 instead of byte repetition

The previous approach repeated secret bytes in a cycle, producing
a weak AES key for short secrets. Now uses SHA-256 with a domain
separator to derive a full-entropy 32-byte block key.
Note: invalidates existing sessions (users must re-login)."
```

---

## Task 8: Fix OAuth state comparison to use constant-time compare

`oauth_handlers.go:83` uses `!=` for state comparison, which is vulnerable to timing attacks.

**Files:**
- Modify: `internal/api/oauth_handlers.go:59-89`

- [ ] **Step 1: Use constant-time comparison for state validation**

Add `"crypto/subtle"` to the imports. Replace the state check:

```go
	queryState := r.URL.Query().Get("state")
	if queryState == "" || subtle.ConstantTimeCompare([]byte(queryState), []byte(stateCookie.Value)) != 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_state",
			"message": "OAuth state mismatch",
		})
		return
	}
```

- [ ] **Step 2: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/api/oauth_handlers.go
git commit -m "fix(security): use constant-time comparison for OAuth state

Prevents timing-based side channel on the OAuth state parameter."
```

---

## Task 9: Fix MFA session upgrade race condition

`UpgradeMFA` deletes then recreates the session â€” not atomic. Concurrent requests between delete and create get 401. Add an `UpdateSessionMFAPassed` method to the store instead.

**Files:**
- Modify: `internal/storage/storage.go` (add interface method)
- Modify: `internal/storage/sqlite.go` (add implementation)
- Modify: `internal/auth/session.go:175-194` (simplify UpgradeMFA)

- [ ] **Step 1: Add `UpdateSessionMFAPassed` to the Store interface**

In `internal/storage/storage.go`, add to the Sessions section of the interface:

```go
	UpdateSessionMFAPassed(ctx context.Context, id string, mfaPassed bool) error
```

- [ ] **Step 2: Implement in SQLiteStore**

In `internal/storage/sqlite.go`, add after the `DeleteExpiredSessions` method (find it first, it's after the sessions section):

```go
func (s *SQLiteStore) UpdateSessionMFAPassed(ctx context.Context, id string, mfaPassed bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET mfa_passed = ? WHERE id = ?`,
		boolToInt(mfaPassed), id,
	)
	return err
}
```

- [ ] **Step 3: Simplify UpgradeMFA in session.go**

Replace the `UpgradeMFA` method in `internal/auth/session.go`:

```go
// UpgradeMFA sets mfa_passed=true on a session atomically.
func (sm *SessionManager) UpgradeMFA(ctx context.Context, sessionID string) error {
	return sm.store.UpdateSessionMFAPassed(ctx, sessionID, true)
}
```

- [ ] **Step 4: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/storage/storage.go internal/storage/sqlite.go internal/auth/session.go
git commit -m "fix: atomic MFA session upgrade instead of delete+recreate

UpgradeMFA was deleting and recreating the session, causing a race
condition where concurrent requests between the two operations got
401. Now uses a single UPDATE statement."
```

---

## Task 10: Fix magic link rate limiter memory leak

The per-email `lastSent` map grows unbounded â€” no cleanup goroutine. Add one.

**Files:**
- Modify: `internal/api/magiclink_handlers.go:26-50`

- [ ] **Step 1: Add cleanup goroutine to magic link rate limiter**

Replace the `magicLinkRateLimiter` struct and constructor:

```go
// magicLinkRateLimiter tracks per-email rate limits for magic link sends.
type magicLinkRateLimiter struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	cooldown time.Duration
}

func newMagicLinkRateLimiter(cooldown time.Duration) *magicLinkRateLimiter {
	rl := &magicLinkRateLimiter{
		lastSent: make(map[string]time.Time),
		cooldown: cooldown,
	}
	// Clean up stale entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

// allow returns true if the email is allowed to send a magic link (not rate limited).
func (rl *magicLinkRateLimiter) allow(emailAddr string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	last, ok := rl.lastSent[emailAddr]
	if ok && time.Since(last) < rl.cooldown {
		return false
	}
	rl.lastSent[emailAddr] = time.Now()
	return true
}

// cleanup removes entries older than 2x the cooldown period.
func (rl *magicLinkRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-2 * rl.cooldown)
	for email, last := range rl.lastSent {
		if last.Before(cutoff) {
			delete(rl.lastSent, email)
		}
	}
}
```

- [ ] **Step 2: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/api/magiclink_handlers.go
git commit -m "fix: add cleanup goroutine to magic link rate limiter

The per-email lastSent map grew unbounded. Now cleans up entries
older than 2x the cooldown period every 5 minutes."
```

---

## Task 11: Fix recovery code modulo bias

`mfa.go:164` uses `int(b[i]) % alphabetLen` where `alphabetLen=36`. Since `256 % 36 = 4`, the first 4 chars are slightly more probable. Use rejection sampling.

**Files:**
- Modify: `internal/auth/mfa.go:157-167`

- [ ] **Step 1: Replace modulo with rejection sampling**

Replace the `generateRandomCode` function:

```go
// generateRandomCode generates a random alphanumeric code of the given length
// using rejection sampling to avoid modulo bias.
func generateRandomCode(length int) string {
	alphabetLen := byte(len(recoveryCodeAlphabet))
	// Find the largest multiple of alphabetLen that fits in a byte
	maxValid := byte(256 - (256 % int(alphabetLen))) // 252 for alphabetLen=36
	result := make([]byte, length)
	buf := make([]byte, 1)
	for i := 0; i < length; {
		if _, err := rand.Read(buf); err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		if buf[0] < maxValid {
			result[i] = recoveryCodeAlphabet[buf[0]%alphabetLen]
			i++
		}
		// else: reject and retry
	}
	return string(result)
}
```

- [ ] **Step 2: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/auth/mfa.go
git commit -m "fix(security): eliminate modulo bias in recovery code generation

256 %% 36 = 4, making the first 4 alphabet characters slightly more
probable. Now uses rejection sampling for uniform distribution."
```

---

## Task 12: Add CORS middleware

No CORS configuration exists. Browser-based frontends on different origins can't call the API. Add a configurable CORS middleware.

**Files:**
- Modify: `internal/config/config.go` (add CORS config)
- Create: `internal/api/middleware/cors.go`
- Modify: `internal/api/router.go` (wire up CORS)

- [ ] **Step 1: Add CORS config to server config**

In `internal/config/config.go`, add a field to `ServerConfig`:

```go
type ServerConfig struct {
	Port       int      `koanf:"port"`
	Secret     string   `koanf:"secret"`
	BaseURL    string   `koanf:"base_url"`
	CORSOrigins []string `koanf:"cors_origins"`
}
```

- [ ] **Step 2: Create CORS middleware**

Create `internal/api/middleware/cors.go`:

```go
package middleware

import (
	"net/http"
	"strings"
)

// CORS returns middleware that handles Cross-Origin Resource Sharing.
// If allowedOrigins is empty, CORS headers are not set (same-origin only).
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	originSet := make(map[string]bool, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if allowAll || originSet[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Key")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.Header().Set("Vary", strings.Join([]string{"Origin"}, ", "))
			}

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 3: Wire up CORS in the router**

In `internal/api/router.go`, add CORS middleware right after the global middleware stack (after line 94, before the health check):

```go
	// CORS (must be before route handlers)
	if len(cfg.Server.CORSOrigins) > 0 {
		r.Use(mw.CORS(cfg.Server.CORSOrigins))
	}
```

- [ ] **Step 4: Add default CORS config in config defaults**

In `internal/config/config.go`, the `CORSOrigins` field is a slice, so no default needed â€” it's `nil` by default (CORS disabled). Users opt in via config:

```yaml
server:
  cors_origins:
    - "http://localhost:3000"
    - "https://myapp.com"
```

- [ ] **Step 5: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/api/middleware/cors.go internal/api/router.go
git commit -m "feat: add configurable CORS middleware

No CORS support existed. Browser frontends on different origins
could not call the API. Now supports server.cors_origins config
with explicit origin allowlist. Disabled by default."
```

---

## Task 13: Add cleanup to SSO OIDC state store

The OIDC state store is an in-memory map with no TTL or cleanup â€” unbounded memory growth.

**Files:**
- Modify: `internal/api/sso_handlers.go:14-27`

- [ ] **Step 1: Add TTL and cleanup to the state store**

Replace the `SSOHandlers` struct and constructor:

```go
import (
	// ... existing imports
	"sync"
	"time"
)

// ssoStateEntry holds an OIDC state with an expiry time.
type ssoStateEntry struct {
	connectionID string
	expiresAt    time.Time
}

// SSOHandlers provides HTTP handlers for SSO endpoints.
type SSOHandlers struct {
	manager    *sso.SSOManager
	mu         sync.Mutex
	stateStore map[string]*ssoStateEntry
}

// NewSSOHandlers creates a new SSOHandlers.
func NewSSOHandlers(manager *sso.SSOManager) *SSOHandlers {
	h := &SSOHandlers{
		manager:    manager,
		stateStore: make(map[string]*ssoStateEntry),
	}
	// Clean up expired states every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			h.cleanupStates()
		}
	}()
	return h
}

func (h *SSOHandlers) cleanupStates() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for key, entry := range h.stateStore {
		if now.After(entry.expiresAt) {
			delete(h.stateStore, key)
		}
	}
}
```

- [ ] **Step 2: Update state store/read operations**

In `OIDCAuth`, replace the state store write (line 171):

```go
	// Store state -> connectionID mapping for callback verification
	h.mu.Lock()
	h.stateStore[state] = &ssoStateEntry{
		connectionID: connID,
		expiresAt:    time.Now().Add(10 * time.Minute),
	}
	h.mu.Unlock()
```

In `OIDCCallback`, replace the state lookup (lines 202-207):

```go
	// Verify state matches
	h.mu.Lock()
	entry, ok := h.stateStore[state]
	if ok {
		delete(h.stateStore, state)
	}
	h.mu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired state"})
		return
	}

	if entry.connectionID != connID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "state does not match connection"})
		return
	}
```

- [ ] **Step 3: Run the build**

Run: `cd C:/Users/raulg/Desktop/projects/shark && go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/api/sso_handlers.go
git commit -m "fix: add TTL and cleanup to SSO OIDC state store

The in-memory state map had no expiry or cleanup, causing
unbounded memory growth. States now expire after 10 minutes
and are cleaned up every 5 minutes."
```

---

## Update YAML config templates

After all fixes, update both config files to document the new env var convention (`__` for nesting).

**Files:**
- Modify: `sharkauth.yaml`
- Modify: `sharkauth.local.yaml`

- [ ] **Step 1: Update sharkauth.yaml with comments about env var convention**

Add a comment at the top of `sharkauth.yaml`:

```yaml
# SharkAuth Configuration
#
# Values with ${VAR_NAME} are resolved from environment variables.
# Environment variable overrides use SHARKAUTH_ prefix with double-underscore
# for nesting: SHARKAUTH_SMTP__FROM_NAME overrides smtp.from_name
#
# Single-level keys: SHARKAUTH_SMTP__HOST, SHARKAUTH_SMTP__PORT, SHARKAUTH_SMTP__PASSWORD
# Multi-word keys:   SHARKAUTH_SERVER__BASE_URL, SHARKAUTH_AUTH__SESSION_LIFETIME

server:
  port: 8080
  secret: "${SHARKAUTH_SECRET}"
  base_url: "https://auth.myapp.com"
  cors_origins: []
```

- [ ] **Step 2: Update sharkauth.local.yaml with cors_origins**

Add `cors_origins` to the server block in `sharkauth.local.yaml`:

```yaml
server:
  port: 8080
  secret: "dev-secret-change-me-in-production-min-32-chars!!"
  base_url: "http://localhost:8080"
  cors_origins:
    - "http://localhost:3000"
```

- [ ] **Step 3: Commit**

```bash
git add sharkauth.yaml sharkauth.local.yaml
git commit -m "docs: update config templates with env var convention and CORS"
```
