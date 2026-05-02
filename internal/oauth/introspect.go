package oauth

// HandleIntrospect implements RFC 7662 Token Introspection.
// POST /oauth/introspect
//
// Auth: client credentials (Basic or body) OR admin Bearer sk_live_*.
// Returns {"active":true,...claims...} for valid tokens, {"active":false} otherwise.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/storage"
)

// introspectResponse is the RFC 7662 Â§2.2 response object.
// When active is false, no other members are included (marshalled separately).
type introspectResponse struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	Iat       int64  `json:"iat,omitempty"`
	Nbf       int64  `json:"nbf,omitempty"`
	Aud       string `json:"aud,omitempty"`
	Iss       string `json:"iss,omitempty"`
	JTI       string `json:"jti,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	Username  string `json:"username,omitempty"`
}

// HandleIntrospect handles POST /oauth/introspect per RFC 7662.
func (s *Server) HandleIntrospect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeIntrospectError(w, http.StatusBadRequest, "invalid_request", "failed to parse form")
		return
	}

	// Authenticate the caller.
	_, _, err := s.authenticateClient(r)
	if err != nil {
		slog.Debug("introspect: client auth failed", "error", err)
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
		writeIntrospectError(w, http.StatusUnauthorized, "invalid_client", "client authentication required")
		return
	}

	tokenStr := r.FormValue("token")
	if tokenStr == "" {
		// Per RFC 7662 Â§2.1, missing token â†’ active:false (not an error response).
		writeInactiveToken(w)
		return
	}

	// Attempt to find the token in DB. Try JWT path first, then opaque hash.
	dbToken := s.findTokenInDB(ctx, tokenStr)
	if dbToken == nil {
		writeInactiveToken(w)
		return
	}

	// Check revoked.
	if dbToken.RevokedAt != nil {
		writeInactiveToken(w)
		return
	}

	// Check expiry.
	if time.Now().UTC().After(dbToken.ExpiresAt) {
		writeInactiveToken(w)
		return
	}

	// Build the response from the DB record (authoritative source).
	resp := &introspectResponse{
		Active:    true,
		Scope:     dbToken.Scope,
		ClientID:  dbToken.ClientID,
		Sub:       dbToken.UserID,
		Exp:       dbToken.ExpiresAt.Unix(),
		Iat:       dbToken.CreatedAt.Unix(),
		Nbf:       dbToken.CreatedAt.Unix(),
		Iss:       s.Issuer,
		JTI:       dbToken.JTI,
		TokenType: "Bearer",
		AgentID:   dbToken.AgentID,
	}

	// Enrich with audience from the DB record.
	if dbToken.Audience != "" {
		resp.Aud = dbToken.Audience
	}

	// Try to enrich with user email from the DB if we have a user ID.
	if dbToken.UserID != "" {
		if user, err := s.RawStore.GetUserByID(ctx, dbToken.UserID); err == nil {
			resp.Username = user.Email
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp) //#nosec G104
}

// LookupBearer resolves a raw OAuth bearer token (JWT or opaque) to the
// stored OAuthToken record, or nil if no match. Exposed so other packages
// (vault handlers, agent auth, etc.) can share the canonical three-tier
// lookup instead of rolling their own half-measures that miss JWT tokens.
func (s *Server) LookupBearer(ctx context.Context, tokenStr string) *storage.OAuthToken {
	return s.findTokenInDB(ctx, tokenStr)
}

// findTokenInDB resolves a raw token string to an OAuthToken record.
//
// For JWT access tokens (3-part dot-separated): verify the token signature,
// then use the signed JTI claim for the DB lookup.
//
// For opaque HMAC tokens (2-part dot-separated, format "key.sig"): the store
// saves sha256(sig_part). Split on "." and hash the signature part.
//
// Final fallback: hash the full raw token string (covers edge cases).
func (s *Server) findTokenInDB(ctx context.Context, tokenStr string) *storage.OAuthToken {
	// 1. Try JWT path. Never trust a JTI from an unsigned parse; otherwise an
	// attacker can craft an arbitrary JWT-shaped string with a known JTI.
	if strings.Count(tokenStr, ".") == 2 {
		if claims, err := s.parseSubjectJWT(ctx, tokenStr); err == nil {
			if jti, _ := claims["jti"].(string); jti != "" {
				if tok, err := s.RawStore.GetOAuthTokenByJTI(ctx, jti); err == nil {
					return tok
				}
			}
		}
	}

	// 2. Opaque HMAC token: format is "<tokenKey>.<tokenSignature>", both base64url.
	//    The store persists sha256(signature_part) as token_hash.
	parts := strings.SplitN(tokenStr, ".", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		sigHash := sha256.Sum256([]byte(parts[1]))
		sigHashHex := hex.EncodeToString(sigHash[:])
		if tok, err := s.RawStore.GetOAuthTokenByHash(ctx, sigHashHex); err == nil {
			return tok
		}
	}

	// 3. Final fallback: hash the entire raw token string.
	tokenHash := sha256.Sum256([]byte(tokenStr))
	tokenHashHex := hex.EncodeToString(tokenHash[:])
	if tok, err := s.RawStore.GetOAuthTokenByHash(ctx, tokenHashHex); err == nil {
		return tok
	}

	return nil
}

// authenticateClient validates the caller's identity for introspect/revoke endpoints.
// Priority: Admin Bearer (sk_live_*) > HTTP Basic > form params.
// Returns (clientID, isAdmin, error). On admin auth, clientID is "__admin__".
func (s *Server) authenticateClient(r *http.Request) (clientID string, isAdmin bool, err error) {
	ctx := r.Context()

	// 1. Check for admin Bearer token (sk_live_*).
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			rawKey := parts[1]
			if strings.HasPrefix(rawKey, "sk_live_") {
				keyHash := auth.HashAPIKey(rawKey)
				apiKey, apiKeyErr := s.RawStore.GetAPIKeyByKeyHash(ctx, keyHash)
				if apiKeyErr != nil {
					return "", false, errors.New("invalid admin api key")
				}
				if apiKey.RevokedAt != nil {
					return "", false, errors.New("api key revoked")
				}
				// Check expiry.
				if apiKey.ExpiresAt != nil && *apiKey.ExpiresAt != "" {
					if expAt, parseErr := time.Parse(time.RFC3339, *apiKey.ExpiresAt); parseErr == nil {
						if time.Now().UTC().After(expAt) {
							return "", false, errors.New("api key expired")
						}
					}
				}
				var scopes []string
				if err := json.Unmarshal([]byte(apiKey.Scopes), &scopes); err != nil {
					return "", false, errors.New("invalid admin api key scopes")
				}
				if !auth.CheckScope(scopes, "*") {
					return "", false, errors.New("api key lacks admin scope")
				}
				return "__admin__", true, nil
			}
		}
	}

	// 2. HTTP Basic Auth.
	basicClientID, clientSecret, ok := r.BasicAuth()
	if ok && basicClientID != "" {
		return s.validateClientCredentials(ctx, basicClientID, clientSecret)
	}

	// 3. Form params.
	formClientID := r.FormValue("client_id")
	formSecret := r.FormValue("client_secret")
	if formClientID != "" {
		return s.validateClientCredentials(ctx, formClientID, formSecret)
	}

	return "", false, errors.New("no client credentials provided")
}

// validateClientCredentials looks up the agent by clientID and verifies the secret.
func (s *Server) validateClientCredentials(ctx context.Context, clientID, clientSecret string) (string, bool, error) {
	agent, err := s.RawStore.GetAgentByClientID(ctx, clientID)
	if err != nil {
		return "", false, errors.New("invalid client")
	}
	if !agent.Active {
		return "", false, errors.New("client is inactive")
	}
	if !verifyClientSecret(agent.ClientSecretHash, clientSecret) {
		return "", false, errors.New("invalid client secret")
	}
	return clientID, false, nil
}

// writeInactiveToken writes {"active":false} per RFC 7662 Â§2.2.
func writeInactiveToken(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"active":false}`)) //#nosec G104
}

// writeIntrospectError writes an OAuth JSON error response.
func writeIntrospectError(w http.ResponseWriter, status int, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{ //#nosec G104
		"error":             errCode,
		"error_description": description,
	})
}
