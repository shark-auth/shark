package sso_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/sso"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

func strPtr(s string) *string { return &s }

// mockOIDCProvider sets up an httptest.Server that acts as an OIDC identity provider
// with discovery, JWKS, and token endpoints.
type mockOIDCProvider struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
	keyID      string
	lastNonce  string // captured from authorize request for inclusion in ID token
}

func newMockOIDCProvider(t *testing.T) *mockOIDCProvider {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	m := &mockOIDCProvider{
		privateKey: key,
		keyID:      "test-key-1",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", m.discoveryHandler)
	mux.HandleFunc("GET /keys", m.jwksHandler)
	mux.HandleFunc("POST /token", m.tokenHandler)
	mux.HandleFunc("GET /authorize", m.authorizeHandler)

	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockOIDCProvider) close() {
	m.server.Close()
}

func (m *mockOIDCProvider) issuer() string {
	return m.server.URL
}

func (m *mockOIDCProvider) discoveryHandler(w http.ResponseWriter, _ *http.Request) {
	doc := map[string]interface{}{
		"issuer":                 m.server.URL,
		"authorization_endpoint": m.server.URL + "/authorize",
		"token_endpoint":         m.server.URL + "/token",
		"jwks_uri":               m.server.URL + "/keys",
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"subject_types_supported":               []string{"public"},
		"response_types_supported":              []string{"code"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (m *mockOIDCProvider) jwksHandler(w http.ResponseWriter, _ *http.Request) {
	jwk := jose.JSONWebKey{
		Key:       m.privateKey.Public(),
		KeyID:     m.keyID,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

func (m *mockOIDCProvider) authorizeHandler(w http.ResponseWriter, r *http.Request) {
	// Capture nonce so the token endpoint can include it in the ID token
	m.lastNonce = r.URL.Query().Get("nonce")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"state":        r.URL.Query().Get("state"),
		"redirect_uri": r.URL.Query().Get("redirect_uri"),
	})
}

// tokenHandler simulates the OIDC token endpoint that exchanges an auth code for tokens.
// It reads the nonce from the stored lastNonce field to include it in the ID token.
func (m *mockOIDCProvider) tokenHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// Generate an ID token
	signerOpts := jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signerOpts.WithHeader("kid", m.keyID)

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: m.privateKey},
		&signerOpts,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("create signer: %v", err), http.StatusInternalServerError)
		return
	}

	clientID := r.FormValue("client_id")
	if clientID == "" {
		// Try basic auth
		clientID, _, _ = r.BasicAuth()
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer:    m.server.URL,
		Subject:   "test-subject-123",
		Audience:  jwt.Audience{clientID},
		IssuedAt:  jwt.NewNumericDate(now),
		Expiry:    jwt.NewNumericDate(now.Add(1 * time.Hour)),
		NotBefore: jwt.NewNumericDate(now.Add(-1 * time.Minute)),
	}

	extraClaims := map[string]interface{}{
		"email": "sso-user@corp.com",
		"name":  "SSO User",
		"nonce": m.lastNonce,
	}

	rawToken, err := jwt.Signed(signer).Claims(claims).Claims(extraClaims).Serialize()
	if err != nil {
		http.Error(w, fmt.Sprintf("sign token: %v", err), http.StatusInternalServerError)
		return
	}

	tokenResp := map[string]interface{}{
		"access_token": "mock-access-token",
		"token_type":   "Bearer",
		"id_token":     rawToken,
		"expires_in":   3600,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResp)
}

func newTestSSOManager(t *testing.T) (*sso.SSOManager, storage.Store) {
	t.Helper()
	store := testutil.NewTestDB(t)
	cfg := testutil.TestConfig()
	sm := auth.NewSessionManager(store, cfg.Server.Secret, cfg.Auth.SessionLifetimeDuration(), cfg.Server.BaseURL)
	mgr := sso.NewSSOManager(store, sm, cfg)
	return mgr, store
}

func TestConnectionCRUD(t *testing.T) {
	mgr, _ := newTestSSOManager(t)
	ctx := context.Background()

	issuer := "https://idp.corp.com"
	clientID := "client-123"
	clientSecret := "secret-456"

	// Create OIDC connection
	conn := &storage.SSOConnection{
		Type:             "oidc",
		Name:             "Test OIDC",
		Domain:           strPtr("corp.com"),
		OIDCIssuer:       &issuer,
		OIDCClientID:     &clientID,
		OIDCClientSecret: &clientSecret,
	}

	if err := mgr.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}
	if conn.ID == "" {
		t.Fatal("connection ID should not be empty")
	}
	if !hasPrefix(conn.ID, "sso_") {
		t.Fatalf("connection ID should have sso_ prefix, got %q", conn.ID)
	}

	// Get
	got, err := mgr.GetConnection(ctx, conn.ID)
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if got.Name != "Test OIDC" {
		t.Fatalf("expected name %q, got %q", "Test OIDC", got.Name)
	}
	if got.Domain == nil || *got.Domain != "corp.com" {
		t.Fatalf("expected domain %q, got %v", "corp.com", got.Domain)
	}

	// List
	conns, err := mgr.ListConnections(ctx)
	if err != nil {
		t.Fatalf("list connections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}

	// Update
	conn.Name = "Updated OIDC"
	if err := mgr.UpdateConnection(ctx, conn); err != nil {
		t.Fatalf("update connection: %v", err)
	}
	got, _ = mgr.GetConnection(ctx, conn.ID)
	if got.Name != "Updated OIDC" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}

	// Delete
	if err := mgr.DeleteConnection(ctx, conn.ID); err != nil {
		t.Fatalf("delete connection: %v", err)
	}
	conns, _ = mgr.ListConnections(ctx)
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections after delete, got %d", len(conns))
	}
}

func TestRouteByEmail(t *testing.T) {
	mgr, _ := newTestSSOManager(t)
	ctx := context.Background()

	issuer := "https://idp.corp.com"
	clientID := "client-id"

	// Create connection with domain
	conn := &storage.SSOConnection{
		Type:         "oidc",
		Name:         "Corp OIDC",
		Domain:       strPtr("corp.com"),
		OIDCIssuer:   &issuer,
		OIDCClientID: &clientID,
	}
	if err := mgr.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	// Route by email should find connection
	found, err := mgr.RouteByEmail(ctx, "user@corp.com")
	if err != nil {
		t.Fatalf("route by email: %v", err)
	}
	if found.ID != conn.ID {
		t.Fatalf("expected connection %q, got %q", conn.ID, found.ID)
	}

	// Route by unknown domain should fail
	_, err = mgr.RouteByEmail(ctx, "user@other.com")
	if err == nil {
		t.Fatal("expected error for unknown domain")
	}

	// Invalid email should fail
	_, err = mgr.RouteByEmail(ctx, "invalid")
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestOIDCFlow(t *testing.T) {
	// Set up mock OIDC provider
	idp := newMockOIDCProvider(t)
	defer idp.close()

	mgr, _ := newTestSSOManager(t)
	ctx := context.Background()

	issuerURL := idp.issuer()
	clientID := "test-client"
	clientSecret := "test-secret"

	// Create OIDC connection pointing to mock provider
	conn := &storage.SSOConnection{
		Type:             "oidc",
		Name:             "Mock OIDC",
		Domain:           strPtr("corp.com"),
		OIDCIssuer:       &issuerURL,
		OIDCClientID:     &clientID,
		OIDCClientSecret: &clientSecret,
	}
	if err := mgr.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	// Step 1: Begin OIDC auth
	redirectURL, state, nonce, err := mgr.BeginOIDCAuth(ctx, conn.ID)
	if err != nil {
		t.Fatalf("begin oidc auth: %v", err)
	}
	if redirectURL == "" {
		t.Fatal("redirect URL should not be empty")
	}
	if state == "" {
		t.Fatal("state should not be empty")
	}
	if nonce == "" {
		t.Fatal("nonce should not be empty")
	}

	// The redirect URL should point to the mock provider's authorize endpoint
	t.Logf("redirect URL: %s", redirectURL)

	// Set the nonce on the mock provider so the token endpoint includes it in the ID token
	idp.lastNonce = nonce

	// Step 2: Handle callback (simulate code exchange)
	// The mock provider's token endpoint returns a valid ID token with the nonce
	mockReq := httptest.NewRequest("GET", "/callback?code=mock-auth-code&state="+state, nil)
	user, session, err := mgr.HandleOIDCCallback(ctx, conn.ID, "mock-auth-code", state, nonce, mockReq)
	if err != nil {
		t.Fatalf("handle oidc callback: %v", err)
	}

	if user == nil {
		t.Fatal("user should not be nil")
	}
	if user.Email != "sso-user@corp.com" {
		t.Fatalf("expected email %q, got %q", "sso-user@corp.com", user.Email)
	}
	if user.Name == nil || *user.Name != "SSO User" {
		t.Fatalf("expected name %q, got %v", "SSO User", user.Name)
	}
	if !user.EmailVerified {
		t.Fatal("SSO user should have email verified")
	}

	if session == nil {
		t.Fatal("session should not be nil")
	}
	if session.AuthMethod != "sso" {
		t.Fatalf("expected auth_method %q, got %q", "sso", session.AuthMethod)
	}
	if session.UserID != user.ID {
		t.Fatalf("session user_id mismatch: %q != %q", session.UserID, user.ID)
	}

	// Step 3: Verify SSO identity was created and repeat login links to same user
	user2, session2, err := mgr.HandleOIDCCallback(ctx, conn.ID, "mock-auth-code", state, nonce, mockReq)
	if err != nil {
		t.Fatalf("second callback: %v", err)
	}
	if user2.ID != user.ID {
		t.Fatalf("expected same user on repeat login, got %q vs %q", user.ID, user2.ID)
	}
	if session2.ID == session.ID {
		t.Fatal("expected different session on repeat login")
	}
}

func TestOIDCFlow_InvalidConnectionType(t *testing.T) {
	mgr, _ := newTestSSOManager(t)
	ctx := context.Background()

	idpURL := "https://idp.example.com/sso"
	conn := &storage.SSOConnection{
		Type:       "saml",
		Name:       "SAML Conn",
		SAMLIdPURL: &idpURL,
	}
	if err := mgr.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	_, _, _, err := mgr.BeginOIDCAuth(ctx, conn.ID)
	if err == nil {
		t.Fatal("expected error when using OIDC flow with SAML connection")
	}
}

func TestOIDCFlow_DisabledConnection(t *testing.T) {
	idp := newMockOIDCProvider(t)
	defer idp.close()

	mgr, _ := newTestSSOManager(t)
	ctx := context.Background()

	issuerURL := idp.issuer()
	clientID := "client"
	clientSecret := "secret"

	conn := &storage.SSOConnection{
		Type:             "oidc",
		Name:             "Disabled OIDC",
		OIDCIssuer:       &issuerURL,
		OIDCClientID:     &clientID,
		OIDCClientSecret: &clientSecret,
	}
	if err := mgr.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	// Disable
	conn.Enabled = false
	if err := mgr.UpdateConnection(ctx, conn); err != nil {
		t.Fatalf("update connection: %v", err)
	}

	_, _, _, err := mgr.BeginOIDCAuth(ctx, conn.ID)
	if err == nil {
		t.Fatal("expected error for disabled connection")
	}
}

func TestSSOHandlers_AutoRoute(t *testing.T) {
	mgr, _ := newTestSSOManager(t)
	ctx := context.Background()

	issuerURL := "https://idp.bigcorp.com"
	clientID := "client"

	// Create connection
	conn := &storage.SSOConnection{
		Type:         "oidc",
		Name:         "Corp SSO",
		Domain:       strPtr("bigcorp.com"),
		OIDCIssuer:   &issuerURL,
		OIDCClientID: &clientID,
	}
	if err := mgr.CreateConnection(ctx, conn); err != nil {
		t.Fatalf("create connection: %v", err)
	}

	// Test auto-route
	routed, err := mgr.RouteByEmail(ctx, "employee@bigcorp.com")
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if routed.ID != conn.ID {
		t.Fatalf("routed to wrong connection")
	}
	if routed.Type != "oidc" {
		t.Fatalf("expected oidc type, got %q", routed.Type)
	}
}

func TestCreateConnection_Validation(t *testing.T) {
	mgr, _ := newTestSSOManager(t)
	ctx := context.Background()

	// Invalid type
	err := mgr.CreateConnection(ctx, &storage.SSOConnection{
		Type: "invalid",
		Name: "Test",
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}

	// Missing name
	err = mgr.CreateConnection(ctx, &storage.SSOConnection{
		Type: "oidc",
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// Ensure the mock provider key is used for signing (suppress unused warning).
var _ crypto.Signer = (*rsa.PrivateKey)(nil)
