package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// AdminAPIKeyFromStore returns a middleware that validates admin access using M2M API keys.
// It checks the Authorization: Bearer sk_live_... header against the api_keys table
// and requires the key to have "*" (wildcard/admin) scope.
// Unlike RequireAPIKey, this middleware does not enforce rate limiting or track last_used_at
// asynchronously, avoiding SQLite lock contention on admin endpoints.
func AdminAPIKeyFromStore(store storage.Store, rateLimiter *auth.TokenBucket) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Read Authorization header
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

			// Hash and look up
			keyHash := auth.HashAPIKey(rawKey)
			apiKey, err := store.GetAPIKeyByKeyHash(r.Context(), keyHash)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid API key")
				return
			}

			// Check revoked
			if apiKey.RevokedAt != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "API key has been revoked")
				return
			}

			// Check expired
			if apiKey.ExpiresAt != nil && *apiKey.ExpiresAt != "" {
				expiresAt, err := time.Parse(time.RFC3339, *apiKey.ExpiresAt)
				if err == nil && time.Now().UTC().After(expiresAt) {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized", "API key has expired")
					return
				}
			}

			// Check scope — admin requires "*"
			var scopes []string
			if err := json.Unmarshal([]byte(apiKey.Scopes), &scopes); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "internal_error", "Invalid key scopes")
				return
			}

			if !auth.CheckScope(scopes, "*") {
				writeJSONError(w, http.StatusForbidden, "forbidden", "API key lacks admin scope")
				return
			}

			// Update last_used_at synchronously (admin requests are low-volume)
			now := time.Now().UTC().Format(time.RFC3339)
			apiKey.LastUsedAt = &now
			_ = store.UpdateAPIKey(r.Context(), apiKey)

			next.ServeHTTP(w, r)
		})
	}
}
