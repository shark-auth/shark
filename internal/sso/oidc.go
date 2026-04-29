package sso

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/shark-auth/shark/internal/storage"
	"golang.org/x/oauth2"
)

// OIDCState holds the state for an in-progress OIDC auth flow.
// In production this should be stored server-side (e.g. in the session store);
// for now we use a simple in-memory map protected by the caller.
type OIDCState struct {
	ConnectionID string
	State        string
	Nonce        string
}

// BeginOIDCAuth constructs the OIDC authorization URL for the given connection.
// It returns the redirect URL, a state token, and a nonce that must be verified in the callback.
func (s *SSOManager) BeginOIDCAuth(ctx context.Context, connectionID string) (redirectURL, state, nonce string, err error) {
	conn, err := s.GetConnection(ctx, connectionID)
	if err != nil {
		return "", "", "", fmt.Errorf("get connection: %w", err)
	}
	if conn.Type != "oidc" {
		return "", "", "", fmt.Errorf("connection %q is not OIDC (type=%s)", connectionID, conn.Type)
	}
	if !conn.Enabled {
		return "", "", "", fmt.Errorf("connection %q is disabled", connectionID)
	}

	provider, err := oidc.NewProvider(ctx, derefStr(conn.OIDCIssuer))
	if err != nil {
		return "", "", "", fmt.Errorf("oidc discovery for issuer %q: %w", derefStr(conn.OIDCIssuer), err)
	}

	oauth2Cfg := s.oidcOAuth2Config(conn, provider)

	stateToken, err := randomToken(16)
	if err != nil {
		return "", "", "", fmt.Errorf("generate state: %w", err)
	}

	nonceToken, err := randomToken(16)
	if err != nil {
		return "", "", "", fmt.Errorf("generate nonce: %w", err)
	}

	url := oauth2Cfg.AuthCodeURL(stateToken, oidc.Nonce(nonceToken))

	return url, stateToken, nonceToken, nil
}

// HandleOIDCCallback exchanges the authorization code, verifies the ID token
// (including nonce validation), creates/links the user, and creates a session.
func (s *SSOManager) HandleOIDCCallback(ctx context.Context, connectionID, code, state, expectedNonce string, r *http.Request) (*storage.User, *storage.Session, error) {
	conn, err := s.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, nil, fmt.Errorf("get connection: %w", err)
	}
	if conn.Type != "oidc" {
		return nil, nil, fmt.Errorf("connection %q is not OIDC", connectionID)
	}

	provider, err := oidc.NewProvider(ctx, derefStr(conn.OIDCIssuer))
	if err != nil {
		return nil, nil, fmt.Errorf("oidc discovery: %w", err)
	}

	oauth2Cfg := s.oidcOAuth2Config(conn, provider)

	// Exchange authorization code for tokens
	token, err := oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("exchange code: %w", err)
	}

	// Extract and verify the ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, nil, fmt.Errorf("no id_token in token response")
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: derefStr(conn.OIDCClientID)})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, nil, fmt.Errorf("verify id_token: %w", err)
	}

	// Validate nonce to prevent replay attacks
	if idToken.Nonce != expectedNonce {
		return nil, nil, fmt.Errorf("id_token nonce mismatch")
	}

	// Extract claims
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, nil, fmt.Errorf("parse id_token claims: %w", err)
	}
	if claims.Sub == "" {
		return nil, nil, fmt.Errorf("id_token missing sub claim")
	}
	if claims.Email == "" {
		return nil, nil, fmt.Errorf("id_token missing email claim")
	}

	// Find or create user, link SSO identity
	user, err := s.findOrCreateUser(ctx, connectionID, claims.Sub, claims.Email, claims.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("find or create user: %w", err)
	}

	// Create session with SSO auth method
	ip := r.RemoteAddr
	ua := r.UserAgent()
	session, err := s.sessions.CreateSession(ctx, user.ID, ip, ua, "sso")
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return user, session, nil
}

// oidcOAuth2Config builds the oauth2 config for an OIDC connection.
func (s *SSOManager) oidcOAuth2Config(conn *storage.SSOConnection, provider *oidc.Provider) oauth2.Config {
	callbackURL := fmt.Sprintf("%s/api/v1/sso/oidc/%s/callback", s.cfg.Server.BaseURL, conn.ID)
	return oauth2.Config{
		ClientID:     derefStr(conn.OIDCClientID),
		ClientSecret: derefStr(conn.OIDCClientSecret),
		Endpoint:     provider.Endpoint(),
		RedirectURL:  callbackURL,
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
