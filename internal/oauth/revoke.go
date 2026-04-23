package oauth

// HandleRevoke implements RFC 7009 Token Revocation.
// POST /oauth/revoke
//
// Auth: client credentials (Basic or body) OR admin Bearer sk_live_*.
// ALWAYS returns 200 OK per RFC 7009 §2.2, even for unknown/invalid tokens.

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HandleRevoke handles POST /oauth/revoke per RFC 7009.
func (s *Server) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		// Even parse errors: per RFC 7009 §2.2, return 200 with empty body.
		// Exception: if the request is malformed enough that we can't parse form,
		// respond 400 (implementation choice, since we can't authenticate).
		writeRevokeError(w, http.StatusBadRequest, "invalid_request", "failed to parse form")
		return
	}

	// Authenticate the caller.
	callerClientID, isAdmin, err := s.authenticateClient(r)
	if err != nil {
		slog.Debug("revoke: client auth failed", "error", err)
		w.Header().Set("WWW-Authenticate", `Basic realm="oauth"`)
		writeRevokeError(w, http.StatusUnauthorized, "invalid_client", "client authentication required")
		return
	}

	tokenStr := r.FormValue("token")
	if tokenStr == "" {
		// Per RFC 7009 §2.2, empty/missing token is not an error.
		writeRevokeOK(w)
		return
	}

	// Locate the token in the DB.
	dbToken := s.findTokenInDB(ctx, tokenStr)
	if dbToken == nil {
		// Per RFC 7009 §2.2: "if the token passed to the request is not valid...
		// the authorization server responds with HTTP status code 200".
		writeRevokeOK(w)
		return
	}

	// Authorization check: a client may only revoke its own tokens.
	// Admin can revoke any token.
	if !isAdmin && dbToken.ClientID != callerClientID {
		slog.Info("revoke: client attempted to revoke another client's token",
			"caller", callerClientID,
			"token_client", dbToken.ClientID,
		)
		// Per RFC 7009 §2.2, return 200 but do not revoke (no-op).
		writeRevokeOK(w)
		return
	}

	// Already revoked: idempotent — return 200.
	if dbToken.RevokedAt != nil {
		writeRevokeOK(w)
		return
	}

	// Revoke the token.
	if err := s.RawStore.RevokeOAuthToken(ctx, dbToken.ID); err != nil {
		slog.Error("revoke: failed to revoke token", "error", err, "jti", dbToken.JTI)
		writeRevokeError(w, http.StatusInternalServerError, "server_error", "failed to revoke token")
		return
	}

	// If this is a refresh token, revoke the entire family.
	if dbToken.TokenType == "refresh" && dbToken.FamilyID != "" {
		if n, famErr := s.RawStore.RevokeOAuthTokenFamily(ctx, dbToken.FamilyID); famErr != nil {
			slog.Warn("revoke: failed to revoke token family", "error", famErr, "family_id", dbToken.FamilyID)
		} else {
			slog.Info("revoke: revoked refresh token family", "family_id", dbToken.FamilyID, "count", n)
		}
	}

	// If this is an access token and has a family_id, also revoke sibling refresh tokens.
	if dbToken.TokenType == "access" && dbToken.FamilyID != "" {
		if n, famErr := s.RawStore.RevokeOAuthTokenFamily(ctx, dbToken.FamilyID); famErr != nil {
			slog.Warn("revoke: failed to revoke sibling tokens", "error", famErr, "family_id", dbToken.FamilyID)
		} else {
			slog.Info("revoke: revoked sibling token family", "family_id", dbToken.FamilyID, "count", n)
		}
	}

	// Audit.
	slog.Info("oauth.token.revoked",
		"client_id", callerClientID,
		"token_client_id", dbToken.ClientID,
		"jti", dbToken.JTI,
		"token_type", dbToken.TokenType,
		"is_admin", isAdmin,
	)

	writeRevokeOK(w)
}

// writeRevokeOK writes a 200 OK with empty body, per RFC 7009 §2.2.
func writeRevokeOK(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

// writeRevokeError writes an OAuth JSON error response for revocation.
func writeRevokeError(w http.ResponseWriter, status int, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{ //#nosec G104
		"error":             errCode,
		"error_description": description,
	})
}
