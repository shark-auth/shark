package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/storage"
)

const (
	// APIKeyIDKey is the context key for the authenticated API key ID.
	APIKeyIDKey contextKey = "api_key_id"
	// APIKeyScopesKey is the context key for the API key's scopes.
	APIKeyScopesKey contextKey = "api_key_scopes" //#nosec G101 -- context-key identifier string, not a credential
	// APIKeyNameKey is the context key for the API key's name.
	APIKeyNameKey contextKey = "api_key_name"
)

// GetAPIKeyID returns the API key ID from the request context.
func GetAPIKeyID(ctx context.Context) string {
	if v, ok := ctx.Value(APIKeyIDKey).(string); ok {
		return v
	}
	return ""
}

// GetAPIKeyScopes returns the API key's scopes from the request context.
func GetAPIKeyScopes(ctx context.Context) []string {
	if v, ok := ctx.Value(APIKeyScopesKey).([]string); ok {
		return v
	}
	return nil
}

// RequireAPIKey returns middleware that validates Bearer API keys (sk_live_...).
// It:
//  1. Reads Authorization: Bearer sk_live_xxx
//  2. Hashes the key with SHA-256
//  3. Looks up key_hash in the api_keys table
//  4. Verifies not revoked (revoked_at IS NULL)
//  5. Verifies not expired
//  6. Checks scope against the required action
//  7. Enforces rate limit via the provided TokenBucket
//  8. Updates last_used_at asynchronously
//  9. Sets key info in request context
func RequireAPIKey(store storage.Store, rateLimiter *auth.TokenBucket, requiredScope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Read Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid Authorization header format")
				return
			}

			rawKey := parts[1]
			if !strings.HasPrefix(rawKey, "sk_live_") {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid API key format")
				return
			}

			// 2. Hash the key
			keyHash := auth.HashAPIKey(rawKey)

			// 3. Look up by hash
			apiKey, err := store.GetAPIKeyByKeyHash(r.Context(), keyHash)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid API key")
				return
			}

			// 4. Check revoked
			if apiKey.RevokedAt != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "API key has been revoked")
				return
			}

			// 5. Check expired
			if apiKey.ExpiresAt != nil && *apiKey.ExpiresAt != "" {
				expiresAt, err := time.Parse(time.RFC3339, *apiKey.ExpiresAt)
				if err == nil && time.Now().UTC().After(expiresAt) {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized", "API key has expired")
					return
				}
			}

			// 6. Check scope
			var scopes []string
			if err := json.Unmarshal([]byte(apiKey.Scopes), &scopes); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "internal_error", "Invalid key scopes")
				return
			}

			if requiredScope != "" && !auth.CheckScope(scopes, requiredScope) {
				writeJSONError(w, http.StatusForbidden, "forbidden", "API key lacks required scope: "+requiredScope)
				return
			}

			// 7. Rate limit
			if rateLimiter != nil && !rateLimiter.Allow(keyHash, apiKey.RateLimit) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{ //#nosec G104 -- write to ResponseWriter; no actionable recovery
					"error":   "rate_limited",
					"message": "API key rate limit exceeded",
				})
				return
			}

			// 8. Update last_used_at (fire-and-forget, bounded).
			// Intentionally detached from r.Context(): this must complete even
			// if the caller's connection dropped mid-request. Bounded to 5s so
			// the goroutine can't leak on shutdown.
			go func() { //#nosec G118 -- fire-and-forget last-used update; decoupled from request by design, bounded via WithTimeout
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				now := time.Now().UTC().Format(time.RFC3339)
				apiKey.LastUsedAt = &now
				_ = store.UpdateAPIKey(ctx, apiKey)
			}()

			// 9. Set context
			ctx := r.Context()
			ctx = context.WithValue(ctx, APIKeyIDKey, apiKey.ID)
			ctx = context.WithValue(ctx, APIKeyScopesKey, scopes)
			ctx = context.WithValue(ctx, APIKeyNameKey, apiKey.Name)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeJSONError writes a W18-shaped JSON error response. It mirrors
// api.ErrorResponse but is duplicated here because middleware cannot import
// its parent package (internal/api) without a cycle.
//
// Shape: {error, message, code, docs_url}. `code` defaults to `error` so
// switching on `code` works even before callers fan out into finer slugs.
func writeJSONError(w http.ResponseWriter, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{ //#nosec G104 -- write to ResponseWriter; no actionable recovery
		"error":    errCode,
		"message":  message,
		"code":     errCode,
		"docs_url": "https://docs.shark-auth.com/errors/" + errCode,
	})
}
