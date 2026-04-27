package oauth

// HandleTokenExchange implements RFC 8693 Token Exchange for agent-to-agent
// delegation. Grant type: urn:ietf:params:oauth:grant-type:token-exchange
//
// The acting agent authenticates with its own client credentials, presents a
// subject_token (a JWT issued by this server), and receives a new JWT with an
// "act" claim recording the full delegation chain.

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

const (
	grantTypeTokenExchange = "urn:ietf:params:oauth:grant-type:token-exchange"
	tokenTypeAccessToken   = "urn:ietf:params:oauth:token-type:access_token"
)

// HandleTokenExchange handles the token-exchange grant type per RFC 8693.
func (s *Server) HandleTokenExchange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeExchangeError(w, http.StatusBadRequest, "invalid_request", "failed to parse form")
		return
	}

	// --- Step 1: Authenticate the acting agent ---
	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}
	if clientID == "" {
		writeExchangeError(w, http.StatusUnauthorized, "invalid_client", "client authentication required")
		return
	}

	actingAgent, err := s.RawStore.GetAgentByClientID(ctx, clientID)
	if err != nil {
		writeExchangeError(w, http.StatusUnauthorized, "invalid_client", "client not found")
		return
	}
	if !actingAgent.Active {
		writeExchangeError(w, http.StatusUnauthorized, "invalid_client", "client is inactive")
		return
	}
	if !verifyClientSecret(actingAgent.ClientSecretHash, clientSecret) {
		writeExchangeError(w, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return
	}

	// --- Step 2: Parse and validate the subject_token ---
	subjectToken := r.FormValue("subject_token")
	subjectTokenType := r.FormValue("subject_token_type")
	if subjectToken == "" {
		writeExchangeError(w, http.StatusBadRequest, "invalid_request", "subject_token is required")
		return
	}
	if subjectTokenType == "" {
		writeExchangeError(w, http.StatusBadRequest, "invalid_request", "subject_token_type is required")
		return
	}
	if subjectTokenType != tokenTypeAccessToken {
		writeExchangeError(w, http.StatusBadRequest, "invalid_request", "unsupported subject_token_type")
		return
	}

	subjectClaims, err := s.parseSubjectJWT(ctx, subjectToken)
	if err != nil {
		slog.Debug("token_exchange: invalid subject_token", "error", err)
		writeExchangeError(w, http.StatusBadRequest, "invalid_token", "subject_token is invalid or expired")
		return
	}

	// Check whether the subject token JTI is revoked.
	if subjectJTI, _ := subjectClaims["jti"].(string); subjectJTI != "" {
		if revoked, rErr := s.RawStore.IsRevokedJTI(ctx, subjectJTI); rErr == nil && revoked {
			writeExchangeError(w, http.StatusBadRequest, "invalid_token", "subject_token has been revoked")
			return
		}
	}

	subjectSub, _ := subjectClaims["sub"].(string)
	subjectScope, _ := subjectClaims["scope"].(string)
	subjectAct, _ := subjectClaims["act"].(map[string]interface{})

	// --- Step 3: Validate scope narrowing ---
	requestedScopeStr := r.FormValue("scope")
	subjectScopes := splitScopes(subjectScope)
	var grantedScopes []string

	if requestedScopeStr != "" {
		requestedScopes := splitScopes(requestedScopeStr)
		if !scopesSubset(requestedScopes, subjectScopes) {
			writeExchangeError(w, http.StatusBadRequest, "invalid_scope", "requested scope exceeds subject_token scope")
			return
		}
		grantedScopes = requestedScopes
	} else {
		grantedScopes = subjectScopes
	}

	// --- Step 4: Check may_act (permissive if absent) ---
	if mayAct, hasMayAct := subjectClaims["may_act"]; hasMayAct {
		if !isMayActAllowed(mayAct, actingAgent.ClientID) {
			writeExchangeError(w, http.StatusForbidden, "access_denied", "acting agent is not permitted by may_act")
			return
		}
	}

	// --- Step 5: Build delegation chain ---
	actClaim := buildActClaim(actingAgent.ClientID, subjectAct)

	// Determine audience / resource.
	audience := r.FormValue("audience")
	if audience == "" {
		audience = r.FormValue("resource")
	}

	// --- Step 6: Issue new JWT ---
	jti, err := gonanoid.New(21)
	if err != nil {
		writeExchangeError(w, http.StatusInternalServerError, "server_error", "failed to generate token ID")
		return
	}

	lifespan := s.Config.AccessTokenLifetimeDuration()
	now := time.Now().UTC()
	expiry := now.Add(lifespan)

	claims := gojwt.MapClaims{
		"iss":        s.Issuer,
		"sub":        subjectSub,
		"client_id":  actingAgent.ClientID,
		"scope":      strings.Join(grantedScopes, " "),
		"act":        actClaim,
		"iat":        now.Unix(),
		"exp":        expiry.Unix(),
		"jti":        jti,
		"agent_id":   actingAgent.ID,
		"agent_name": actingAgent.Name,
	}
	if audience != "" {
		claims["aud"] = audience
	}

	signedToken, err := s.Sign(claims)
	if err != nil {
		slog.Error("token_exchange: failed to sign token", "error", err)
		writeExchangeError(w, http.StatusInternalServerError, "server_error", "failed to sign token")
		return
	}

	// --- Step 7: Store token ---
	// Resolve the subject to a real user ID only if it exists in the users
	// table. When the subject token was a client_credentials token the sub
	// claim holds a client_id (not a user ID), and the oauth_tokens.user_id
	// column has a FK → users(id). Setting it to a non-existent ID triggers a
	// FK violation, so we leave it NULL for agent-to-agent delegation.
	resolvedUserID := ""
	if subjectSub != "" {
		if _, lookupErr := s.RawStore.GetUserByID(ctx, subjectSub); lookupErr == nil {
			resolvedUserID = subjectSub
		}
	}
	tokenHash := hashTokenString(signedToken)
	oauthToken := &storage.OAuthToken{
		ID:                "tok_" + jti[:8],
		JTI:               jti,
		ClientID:          actingAgent.ClientID,
		AgentID:           actingAgent.ID,
		UserID:            resolvedUserID,
		TokenType:         "access",
		TokenHash:         tokenHash,
		Scope:             strings.Join(grantedScopes, " "),
		Audience:          audience,
		DelegationSubject: subjectSub,
		DelegationActor:   actingAgent.ID,
		ExpiresAt:         expiry,
		CreatedAt:         now,
	}
	if err := s.RawStore.CreateOAuthToken(ctx, oauthToken); err != nil {
		slog.Error("token_exchange: failed to store token", "error", err)
		writeExchangeError(w, http.StatusInternalServerError, "server_error", "failed to store token")
		return
	}

	// Compute dropped scopes: subject_scope - granted_scope.
	requestedScopes := splitScopes(requestedScopeStr)
	droppedScopes := make([]string, 0)
	grantedSet := make(map[string]bool, len(grantedScopes))
	for _, sc := range grantedScopes {
		grantedSet[sc] = true
	}
	for _, sc := range subjectScopes {
		if !grantedSet[sc] {
			droppedScopes = append(droppedScopes, sc)
		}
	}

	// Audit.
	actChainJSON, _ := json.Marshal(actClaim)
	slog.Info("oauth.token.exchanged",
		"acting_agent", actingAgent.ClientID,
		"subject", subjectSub,
		"scope", strings.Join(grantedScopes, " "),
		"act_chain", string(actChainJSON),
	)
	if s.AuditLogger != nil {
		// Ensure nil slices serialize as [] not null.
		subjectScopeOut := subjectScopes
		if subjectScopeOut == nil {
			subjectScopeOut = []string{}
		}
		grantedScopeOut := grantedScopes
		if grantedScopeOut == nil {
			grantedScopeOut = []string{}
		}
		if droppedScopes == nil {
			droppedScopes = []string{}
		}
		requestedScopeOut := requestedScopes
		if requestedScopeOut == nil {
			requestedScopeOut = []string{}
		}
		metaMap := map[string]any{
			"act_chain":        actClaim,
			"scope":            strings.Join(grantedScopes, " "),
			"client_id":        actingAgent.ClientID,
			"subject":          subjectSub,
			"subject_scope":    subjectScopeOut,
			"granted_scope":    grantedScopeOut,
			"dropped_scope":    droppedScopes,
			"requested_scope":  requestedScopeOut,
		}
		metaJSON, _ := json.Marshal(metaMap)
		_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
			Action:     "oauth.token.exchanged",
			ActorID:    actingAgent.ID,
			ActorType:  "agent",
			TargetID:   subjectSub,
			TargetType: "token",
			Status:     "success",
			Metadata:   string(metaJSON),
		})
	}

	// RFC 8693 section 2.2.1 response.
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":      signedToken,
		"issued_token_type": tokenTypeAccessToken,
		"token_type":        "Bearer",
		"expires_in":        int64(lifespan.Seconds()),
		"scope":             strings.Join(grantedScopes, " "),
	})
}

// Sign signs MapClaims with the server active ES256 private key and sets the
// kid header. Used by token exchange and any other in-process JWT issuance.
func (s *Server) Sign(claims gojwt.MapClaims) (string, error) {
	if s.signingPrivKey == nil {
		return "", fmt.Errorf("signing private key not loaded on server")
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodES256, claims)
	token.Header["kid"] = s.SigningKeyID
	return token.SignedString(s.signingPrivKey)
}

// parseSubjectJWT parses and validates a JWT against the server's ES256 signing
// key. The key is resolved in two steps:
//
//  1. If the JWT header contains a "kid" claim, the key is looked up by KID
//     (allowing retired keys so that tokens signed before a key rotation are
//     still accepted until they expire).
//  2. Otherwise (no kid), fall back to the current active ES256 key.
func (s *Server) parseSubjectJWT(ctx context.Context, tokenStr string) (gojwt.MapClaims, error) {
	parsed, err := gojwt.ParseWithClaims(tokenStr, gojwt.MapClaims{}, func(t *gojwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		// Prefer kid-based lookup so tokens signed with a recently-rotated (but
		// not yet expired) key remain verifiable after admin key rotation.
		if kid, ok := t.Header["kid"].(string); ok && kid != "" {
			key, lookupErr := s.RawStore.GetSigningKeyByKID(ctx, kid)
			if lookupErr == nil {
				return parseECPublicKeyPEM(key.PublicKeyPEM)
			}
			// KID not in DB — fall through to active-key lookup.
		}
		// No kid or kid not found: try the current active ES256 key.
		key, err := s.RawStore.GetActiveSigningKeyByAlgorithm(ctx, "ES256")
		if err != nil {
			return nil, fmt.Errorf("get active ES256 key: %w", err)
		}
		return parseECPublicKeyPEM(key.PublicKeyPEM)
	})
	if err != nil {
		return nil, fmt.Errorf("parse JWT: %w", err)
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("invalid JWT")
	}
	claims, ok := parsed.Claims.(gojwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type")
	}
	return claims, nil
}

// parseECPublicKeyPEM decodes a PEM-encoded ECDSA public key (PKIX).
func parseECPublicKeyPEM(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA")
	}
	return ecPub, nil
}

// ---------------------------------------------------------------------------
// Pure helpers
// ---------------------------------------------------------------------------

// scopesSubset returns true when every scope in requested also appears in available.
func scopesSubset(requested, available []string) bool {
	availSet := make(map[string]bool, len(available))
	for _, s := range available {
		availSet[s] = true
	}
	for _, r := range requested {
		if !availSet[r] {
			return false
		}
	}
	return true
}

// buildActClaim constructs the RFC 8693 act claim for the delegation chain.
// If the subject token already has an act claim it is nested inside.
func buildActClaim(actingAgentID string, subjectAct map[string]interface{}) map[string]interface{} {
	act := map[string]interface{}{"sub": actingAgentID}
	if subjectAct != nil {
		act["act"] = subjectAct
	}
	return act
}

// splitScopes splits a space-separated scope string into individual tokens.
func splitScopes(scope string) []string {
	return strings.Fields(scope)
}

// isMayActAllowed returns true when actingClientID is listed in the may_act
// claim. may_act may be a single string or a JSON array of strings.
func isMayActAllowed(mayAct interface{}, actingClientID string) bool {
	switch v := mayAct.(type) {
	case string:
		return v == actingClientID
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == actingClientID {
				return true
			}
		}
	}
	return false
}

// verifyClientSecret compares secret against a stored SHA-256 hex hash in
// constant time.
func verifyClientSecret(storedHash, secret string) bool {
	if storedHash == "" || secret == "" {
		return false
	}
	h := sha256.Sum256([]byte(secret))
	computed := hex.EncodeToString(h[:])
	if len(storedHash) != len(computed) {
		return false
	}
	var diff byte
	for i := range storedHash {
		diff |= storedHash[i] ^ computed[i]
	}
	return diff == 0
}

// hashTokenString returns the SHA-256 hex digest of the raw signed JWT string.
func hashTokenString(tokenStr string) string {
	h := sha256.Sum256([]byte(tokenStr))
	return hex.EncodeToString(h[:])
}

// writeExchangeError writes an RFC 6749 JSON error response.
func writeExchangeError(w http.ResponseWriter, status int, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}
