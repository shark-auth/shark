package testutil

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/api"
	"github.com/sharkauth/sharkauth/internal/auth"
	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// TestServer wraps an httptest.Server with a pre-configured HTTP client that
// handles cookies automatically. Provides helper methods for JSON requests.
type TestServer struct {
	Server      *httptest.Server
	Client      *http.Client
	Store       storage.Store
	Config      *config.Config
	T           *testing.T
	APIServer   *api.Server
	EmailSender *MemoryEmailSender
	AdminKey    string // Full admin API key (sk_live_...) for Bearer auth
}

// seedTestDefaultApp inserts a default application into the store if one doesn't exist.
// Seeds the MagicLink.RedirectURL and Social.RedirectURL from config into the allowlist
// so redirect-validation tests pass the same way they will in production.
func seedTestDefaultApp(t *testing.T, store storage.Store, cfg *config.Config) {
	t.Helper()
	ctx := context.Background()

	// Idempotent: if already seeded, skip.
	if _, err := store.GetDefaultApplication(ctx); err == nil {
		return
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("seedTestDefaultApp: rand: %v", err)
	}
	// Simple hex secret for test purposes.
	secret := hex.EncodeToString(b)
	h := sha256.Sum256([]byte(secret))
	secretHash := hex.EncodeToString(h[:])
	secretPrefix := secret
	if len(secretPrefix) > 8 {
		secretPrefix = secretPrefix[:8]
	}

	nid, _ := gonanoid.New(21)
	appNid, _ := gonanoid.New()
	now := time.Now().UTC()

	var callbacks []string
	if cfg.Social.RedirectURL != "" {
		callbacks = append(callbacks, cfg.Social.RedirectURL)
	}
	if cfg.MagicLink.RedirectURL != "" {
		dup := false
		for _, u := range callbacks {
			if u == cfg.MagicLink.RedirectURL {
				dup = true
				break
			}
		}
		if !dup {
			callbacks = append(callbacks, cfg.MagicLink.RedirectURL)
		}
	}
	if callbacks == nil {
		callbacks = []string{}
	}

	app := &storage.Application{
		ID:                  "app_" + appNid,
		Name:                "Default Application",
		ClientID:            "shark_app_" + nid,
		ClientSecretHash:    secretHash,
		ClientSecretPrefix:  secretPrefix,
		AllowedCallbackURLs: callbacks,
		AllowedLogoutURLs:   []string{},
		AllowedOrigins:      []string{},
		IsDefault:           true,
		Metadata:            map[string]any{},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := store.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seedTestDefaultApp: create: %v", err)
	}
}

// newTestJWTManager creates and provisions a JWT manager backed by the given store.
// Generates an ephemeral signing key so JWT-dependent code works in tests.
func newTestJWTManager(t *testing.T, store storage.Store, cfg *config.Config) *jwtpkg.Manager {
	t.Helper()
	jm := jwtpkg.NewManager(&cfg.Auth.JWT, store, cfg.Server.BaseURL, cfg.Server.Secret)
	if err := jm.EnsureActiveKey(context.Background()); err != nil {
		t.Fatalf("testutil: JWT EnsureActiveKey: %v", err)
	}
	return jm
}

// NewTestServer creates a test HTTP server with all routes mounted.
// The server is automatically closed when the test completes.
// An admin API key with "*" scope is automatically created for test use.
// A JWT manager with an ephemeral signing key is auto-provisioned.
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	store := NewTestDB(t)
	cfg := TestConfig()
	seedTestDefaultApp(t, store, cfg)
	emailSender := NewMemoryEmailSender()
	jm := newTestJWTManager(t, store, cfg)
	srv := api.NewServer(store, cfg, api.WithEmailSender(emailSender), api.WithJWTManager(jm))

	ts := httptest.NewServer(srv.Router)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("creating cookiejar: %v", err)
	}

	client := &http.Client{
		Jar: jar,
		// Don't follow redirects automatically so we can inspect them
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Bootstrap an admin API key for tests
	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("generating admin key: %v", err)
	}
	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	adminKey := &storage.APIKey{
		ID:        "key_" + id,
		Name:      "test-admin",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeySuffix: keySuffix,
		Scopes:    `["*"]`,
		RateLimit: 100000,
		CreatedAt: now,
	}
	if err := store.CreateAPIKey(context.Background(), adminKey); err != nil {
		t.Fatalf("creating test admin key: %v", err)
	}

	t.Cleanup(func() {
		ts.Close()
	})

	return &TestServer{
		Server:      ts,
		Client:      client,
		Store:       store,
		Config:      cfg,
		T:           t,
		APIServer:   srv,
		EmailSender: emailSender,
		AdminKey:    fullKey,
	}
}

// URL returns the full URL for a given path on the test server.
func (ts *TestServer) URL(path string) string {
	return ts.Server.URL + path
}

// PostJSON sends a POST request with a JSON body and returns the response.
func (ts *TestServer) PostJSON(path string, body interface{}) *http.Response {
	ts.T.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.T.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", ts.URL(path), bodyReader)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// PatchJSON sends a PATCH request with a JSON body and returns the response.
func (ts *TestServer) PatchJSON(path string, body interface{}) *http.Response {
	ts.T.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.T.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("PATCH", ts.URL(path), bodyReader)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// Get sends a GET request and returns the response.
func (ts *TestServer) Get(path string) *http.Response {
	ts.T.Helper()

	resp, err := ts.Client.Get(ts.URL(path))
	if err != nil {
		ts.T.Fatalf("sending GET request: %v", err)
	}
	return resp
}

// GetWithAdminKey sends a GET request with the admin Bearer API key header.
func (ts *TestServer) GetWithAdminKey(path string) *http.Response {
	ts.T.Helper()

	req, err := http.NewRequest("GET", ts.URL(path), nil)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+ts.AdminKey)

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// PostJSONWithAdminKey sends a POST request with a JSON body and the admin Bearer API key header.
func (ts *TestServer) PostJSONWithAdminKey(path string, body interface{}) *http.Response {
	ts.T.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.T.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", ts.URL(path), bodyReader)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ts.AdminKey)

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// DecodeJSON reads and decodes the response body into the given target.
func (ts *TestServer) DecodeJSON(resp *http.Response, target interface{}) {
	ts.T.Helper()
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		ts.T.Fatalf("decoding response body: %v", err)
	}
}

// SignupAndVerify creates a test user via signup and immediately marks their email as verified.
// Returns the user ID. Use this when testing endpoints that require email verification.
func (ts *TestServer) SignupAndVerify(email, password, name string) string {
	ts.T.Helper()
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	})
	if resp.StatusCode != 201 {
		ts.T.Fatalf("SignupAndVerify: signup failed with status %d", resp.StatusCode)
	}
	var result map[string]interface{}
	ts.DecodeJSON(resp, &result)
	userID := result["id"].(string)

	// Mark email as verified directly in the store
	user, err := ts.Store.GetUserByID(context.Background(), userID)
	if err != nil {
		ts.T.Fatalf("SignupAndVerify: getting user: %v", err)
	}
	user.EmailVerified = true
	if err := ts.Store.UpdateUser(context.Background(), user); err != nil {
		ts.T.Fatalf("SignupAndVerify: verifying email: %v", err)
	}

	return userID
}

// NewTestServerDev is identical to NewTestServer but enables DevMode and wires
// DevInboxSender so the /admin/dev/* routes are mounted and email flows are
// captured to the dev_emails table.
func NewTestServerDev(t *testing.T) *TestServer {
	t.Helper()

	store := NewTestDB(t)
	cfg := TestConfig()
	cfg.Server.DevMode = true
	seedTestDefaultApp(t, store, cfg)

	dev := email.NewDevInboxSender(store)
	jm := newTestJWTManager(t, store, cfg)
	srv := api.NewServer(store, cfg, api.WithEmailSender(dev), api.WithJWTManager(jm))

	ts := httptest.NewServer(srv.Router)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("generating admin key: %v", err)
	}
	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	adminKey := &storage.APIKey{
		ID: "key_" + id, Name: "test-admin", KeyHash: keyHash,
		KeyPrefix: keyPrefix, KeySuffix: keySuffix,
		Scopes: `["*"]`, RateLimit: 100000, CreatedAt: now,
	}
	if err := store.CreateAPIKey(context.Background(), adminKey); err != nil {
		t.Fatalf("creating test admin key: %v", err)
	}

	t.Cleanup(func() { ts.Close() })

	return &TestServer{
		Server: ts, Client: client, Store: store, Config: cfg, T: t,
		APIServer: srv, AdminKey: fullKey,
	}
}

// NewTestServerWithHandler creates a test HTTP server with the given handler.
// Use this when you need to test a specific handler without the full router.
func NewTestServerWithHandler(t *testing.T, handler http.Handler) *TestServer {
	t.Helper()

	ts := httptest.NewServer(handler)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("creating cookiejar: %v", err)
	}

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Cleanup(func() {
		ts.Close()
	})

	return &TestServer{
		Server: ts,
		Client: client,
		Config: TestConfig(),
		T:      t,
	}
}

// Delete sends a DELETE request and returns the response.
func (ts *TestServer) Delete(path string) *http.Response {
	ts.T.Helper()

	req, err := http.NewRequest("DELETE", ts.URL(path), nil)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// DeleteJSON sends a DELETE request with a JSON body and returns the response.
func (ts *TestServer) DeleteJSON(path string, body interface{}) *http.Response {
	ts.T.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.T.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("DELETE", ts.URL(path), bodyReader)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// DecodeJSONResponse is a generic helper that decodes a JSON response body into
// the specified type T. The response body is closed after reading.
func DecodeJSONResponse[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response body: %v", err)
	}
	return result
}

// DeleteWithAdminKey sends a DELETE request with the admin Bearer API key header.
func (ts *TestServer) DeleteWithAdminKey(path string) *http.Response {
	ts.T.Helper()

	req, err := http.NewRequest("DELETE", ts.URL(path), nil)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+ts.AdminKey)

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// PutJSONWithAdminKey sends a PUT request with a JSON body and the admin Bearer API key header.
func (ts *TestServer) PutJSONWithAdminKey(path string, body interface{}) *http.Response {
	ts.T.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.T.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("PUT", ts.URL(path), bodyReader)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ts.AdminKey)

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// PatchJSONWithAdminKey sends a PATCH request with a JSON body and the admin Bearer API key header.
func (ts *TestServer) PatchJSONWithAdminKey(path string, body interface{}) *http.Response {
	ts.T.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.T.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("PATCH", ts.URL(path), bodyReader)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ts.AdminKey)

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// GetWithBearer sends a GET request with an arbitrary Bearer token.
func (ts *TestServer) GetWithBearer(path, token string) *http.Response {
	ts.T.Helper()

	req, err := http.NewRequest("GET", ts.URL(path), nil)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}

// PostJSONWithBearer sends a POST request with a JSON body and an arbitrary Bearer token.
func (ts *TestServer) PostJSONWithBearer(path string, body interface{}, token string) *http.Response {
	ts.T.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			ts.T.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", ts.URL(path), bodyReader)
	if err != nil {
		ts.T.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("sending request: %v", err)
	}
	return resp
}
