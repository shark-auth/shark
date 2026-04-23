package oauth

// RFC 8707 Resource Indicators tests.
// Covers: client_credentials + resource, authorize-code flow + resource,
// introspection aud field, and the ValidateAudience helper.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ory/fosite"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// doClientCredentials fires a client_credentials token request and returns the
// decoded body + HTTP status.
func doClientCredentials(t *testing.T, ts *httptest.Server, clientID, secret string, extra url.Values) (int, map[string]interface{}) {
	t.Helper()
	form := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"openid"},
	}
	for k, vs := range extra {
		form[k] = vs
	}
	req, err := http.NewRequest("POST", ts.URL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decoding JSON: %v\nbody: %s", err, body)
	}
	return resp.StatusCode, result
}

// ---------------------------------------------------------------------------
// TestResource_ClientCredentials_SingleResource
// ---------------------------------------------------------------------------

// TestResource_ClientCredentials_SingleResource verifies that a
// client_credentials request with resource=https://api.example.com results in
// an access token whose stored audience equals the requested resource.
func TestResource_ClientCredentials_SingleResource(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "res-cc-client", false)

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	const wantResource = "https://api.example.com"

	status, body := doClientCredentials(t, ts, "res-cc-client", "test-secret", url.Values{
		"resource": {wantResource},
	})

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}
	if _, ok := body["access_token"]; !ok {
		t.Fatal("response missing access_token")
	}

	// The stored token should have audience = wantResource.
	// We verify by introspecting the raw DB via the store.
	// Query the store directly for the most recent access token for this client.
	// createTokenSession stores ClientID but not AgentID for fosite-issued tokens,
	// so we use a raw SQL query against the underlying DB.
	ctx := context.Background()
	var foundAud string
	row := store.DB().QueryRowContext(ctx,
		`SELECT audience FROM oauth_tokens WHERE client_id = ? AND token_type = 'access' ORDER BY created_at DESC LIMIT 1`,
		"res-cc-client")
	if err := row.Scan(&foundAud); err != nil {
		t.Fatalf("querying token audience: %v", err)
	}
	if foundAud != wantResource {
		t.Errorf("token audience: want %q, got %q", wantResource, foundAud)
	}
}

// ---------------------------------------------------------------------------
// TestResource_ClientCredentials_NoResource
// ---------------------------------------------------------------------------

// TestResource_ClientCredentials_NoResource verifies that when no resource
// parameter is sent the stored token audience is empty.
func TestResource_ClientCredentials_NoResource(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "res-cc-no-res", false)

	ts := httptest.NewServer(mountTestRouter(srv))
	defer ts.Close()

	status, body := doClientCredentials(t, ts, "res-cc-no-res", "test-secret", nil)

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}

	ctx := context.Background()
	var gotAud string
	row := store.DB().QueryRowContext(ctx,
		`SELECT COALESCE(audience, '') FROM oauth_tokens WHERE client_id = ? AND token_type = 'access' ORDER BY created_at DESC LIMIT 1`,
		"res-cc-no-res")
	if err := row.Scan(&gotAud); err != nil {
		t.Fatalf("querying token audience: %v", err)
	}
	if gotAud != "" {
		t.Errorf("expected empty audience when no resource param, got %q", gotAud)
	}
}

// ---------------------------------------------------------------------------
// TestResource_AuthCodeFlow
// ---------------------------------------------------------------------------

// TestResource_AuthCodeFlow verifies that when resource is included on the
// /oauth/authorize request it gets stored in the authorization code and then
// flows through to the issued access token's audience.
func TestResource_AuthCodeFlow(t *testing.T) {
	fs, store := newTestFositeStore(t)

	seedAgent(t, store, "authcode-res-client", false)
	userID := seedUser(t, store, "res-user@example.com")

	ctx := context.Background()

	const wantResource = "https://resource.example.com"
	const code = "authcode-resource-test-sig"

	client, err := fs.GetClient(ctx, "authcode-res-client")
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}

	// Build a fosite request that includes resource in the form.
	req := buildAuthorizeReq(client, userID, wantResource)

	// Store the authorization code.
	if err := fs.CreateAuthorizeCodeSession(ctx, code, req); err != nil {
		t.Fatalf("CreateAuthorizeCodeSession: %v", err)
	}

	// Verify the authorization code has the resource stored.
	codeHash := hashSignature(code)
	ac, err := store.GetAuthorizationCode(ctx, codeHash)
	if err != nil {
		t.Fatalf("GetAuthorizationCode: %v", err)
	}
	if ac.Resource != wantResource {
		t.Errorf("auth code resource: want %q, got %q", wantResource, ac.Resource)
	}

	// Retrieve the code session (simulates token endpoint reading the code).
	retrieved, err := fs.GetAuthorizeCodeSession(ctx, code, newFositeSession(userID))
	if err != nil {
		t.Fatalf("GetAuthorizeCodeSession: %v", err)
	}

	// The resource must be present in the form of the retrieved request so that
	// createTokenSession can read it.
	if got := retrieved.GetRequestForm().Get("resource"); got != wantResource {
		t.Errorf("retrieved form resource: want %q, got %q", wantResource, got)
	}

	// Now simulate token creation (access token) for this code exchange.
	const tokenSig = "access-sig-authcode-res"
	if err := fs.CreateAccessTokenSession(ctx, tokenSig, retrieved); err != nil {
		t.Fatalf("CreateAccessTokenSession: %v", err)
	}

	// Look up the token and verify audience.
	tokenHash := hashSignature(tokenSig)
	tok, err := store.GetOAuthTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetOAuthTokenByHash: %v", err)
	}
	if tok.Audience != wantResource {
		t.Errorf("access token audience: want %q, got %q", wantResource, tok.Audience)
	}
}

// ---------------------------------------------------------------------------
// TestResource_Introspection_ReturnsAud
// ---------------------------------------------------------------------------

// TestResource_Introspection_ReturnsAud checks that when a token stored with
// an audience is retrieved from the store, the audience is readable from the
// reconstructed request form — which is what the introspection handler uses.
func TestResource_Introspection_ReturnsAud(t *testing.T) {
	fs, store := newTestFositeStore(t)

	seedAgent(t, store, "introspect-res-agent", false)
	userID := seedUser(t, store, "introspect-res@example.com")

	ctx := context.Background()
	const wantAud = "https://api.introspect.example.com"
	const sig = "introspect-aud-sig-123"

	// Seed a token with an audience directly.
	client, _ := fs.GetClient(ctx, "introspect-res-agent")
	req := buildTokenReqWithResource(client, userID, wantAud)

	if err := fs.CreateAccessTokenSession(ctx, sig, req); err != nil {
		t.Fatalf("CreateAccessTokenSession: %v", err)
	}

	// Retrieve and confirm the audience is surfaced via the form.
	retrieved, err := fs.GetAccessTokenSession(ctx, sig, newFositeSession(userID))
	if err != nil {
		t.Fatalf("GetAccessTokenSession: %v", err)
	}
	if got := retrieved.GetRequestForm().Get("resource"); got != wantAud {
		t.Errorf("retrieved resource from form: want %q, got %q", wantAud, got)
	}

	// Also verify it's persisted in the store.
	tokenHash := hashSignature(sig)
	tok, err := store.GetOAuthTokenByHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("GetOAuthTokenByHash: %v", err)
	}
	if tok.Audience != wantAud {
		t.Errorf("stored audience: want %q, got %q", wantAud, tok.Audience)
	}
}

// ---------------------------------------------------------------------------
// TestValidateAudience
// ---------------------------------------------------------------------------

// TestValidateAudience exercises all ValidateAudience branches.
func TestValidateAudience(t *testing.T) {
	cases := []struct {
		name     string
		aud      interface{}
		expected string
		want     bool
	}{
		{"string match", "https://api.example.com", "https://api.example.com", true},
		{"string mismatch", "https://api.example.com", "https://other.example.com", false},
		{"slice match", []string{"https://api.example.com", "https://api2.example.com"}, "https://api2.example.com", true},
		{"slice mismatch", []string{"https://api.example.com"}, "https://other.example.com", false},
		{"interface slice match", []interface{}{"https://api.example.com"}, "https://api.example.com", true},
		{"interface slice mismatch", []interface{}{"https://api.example.com"}, "https://other.example.com", false},
		{"nil aud no expected", nil, "", true},
		{"nil aud with expected", nil, "https://api.example.com", false},
		{"empty expected accepts any", "https://api.example.com", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateAudience(tc.aud, tc.expected)
			if got != tc.want {
				t.Errorf("ValidateAudience(%v, %q) = %v, want %v", tc.aud, tc.expected, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestResource_TokenExchange_ResourceParam (regression guard)
// ---------------------------------------------------------------------------

// TestResource_TokenExchange_ResourceParam confirms that the token-exchange
// handler still accepts the resource parameter (pre-existing behaviour).
func TestResource_TokenExchange_ResourceParam(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "res-exchange-actor", []string{"openid", "read"})

	subjectToken := mintSubjectJWT(t, srv, "res-exchange-user", "openid read", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
		"resource":           {"https://resource-indicator.example.com"},
	}
	status, body := doExchange(t, ts, "res-exchange-actor", "test-secret", form)

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}

	// The issued token should contain aud = the resource.
	tokenStr, _ := body["access_token"].(string)
	parsed, err := srv.parseSubjectJWT(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("parsing issued token: %v", err)
	}
	if !ValidateAudience(parsed["aud"], "https://resource-indicator.example.com") {
		t.Errorf("expected aud to contain resource, got %v", parsed["aud"])
	}
}

// ---------------------------------------------------------------------------
// Local fosite test helpers (avoid importing fosite directly — use aliases)
// ---------------------------------------------------------------------------

func newFositeSession(subject string) fosite.Session {
	return &fosite.DefaultSession{
		Subject: subject,
		ExpiresAt: map[fosite.TokenType]time.Time{
			fosite.AuthorizeCode: time.Now().UTC().Add(10 * time.Minute),
			fosite.AccessToken:   time.Now().UTC().Add(15 * time.Minute),
		},
	}
}

// buildAuthorizeReq builds a fosite.Request with the resource param in the
// form. Used to test CreateAuthorizeCodeSession resource storage.
func buildAuthorizeReq(client fosite.Client, userID, resource string) *fosite.Request {
	form := url.Values{
		"redirect_uri":          {"https://example.com/callback"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
	}
	if resource != "" {
		form.Set("resource", resource)
	}
	return &fosite.Request{
		ID:          "authcode-res-req",
		RequestedAt: time.Now().UTC(),
		Client:      client,
		RequestedScope: fosite.Arguments{"openid"},
		GrantedScope:   fosite.Arguments{"openid"},
		Session: &fosite.DefaultSession{
			Subject: userID,
			ExpiresAt: map[fosite.TokenType]time.Time{
				fosite.AuthorizeCode: time.Now().UTC().Add(10 * time.Minute),
				fosite.AccessToken:   time.Now().UTC().Add(15 * time.Minute),
			},
		},
		Form: form,
	}
}

// buildTokenReqWithResource builds a fosite.Request with resource in the form.
// Used to simulate token creation with an explicit audience.
func buildTokenReqWithResource(client fosite.Client, userID, resource string) *fosite.Request {
	form := url.Values{}
	if resource != "" {
		form.Set("resource", resource)
	}
	return &fosite.Request{
		ID:          "tok-res-req-" + resource,
		RequestedAt: time.Now().UTC(),
		Client:      client,
		RequestedScope: fosite.Arguments{"openid"},
		GrantedScope:   fosite.Arguments{"openid"},
		Session: &fosite.DefaultSession{
			Subject: userID,
			ExpiresAt: map[fosite.TokenType]time.Time{
				fosite.AccessToken: time.Now().UTC().Add(15 * time.Minute),
			},
		},
		Form: form,
	}
}

