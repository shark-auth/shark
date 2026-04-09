package testutil

import (
	"bytes"
	"context"
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
	"github.com/sharkauth/sharkauth/internal/config"
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

// NewTestServer creates a test HTTP server with all routes mounted.
// The server is automatically closed when the test completes.
// An admin API key with "*" scope is automatically created for test use.
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	store := NewTestDB(t)
	cfg := TestConfig()
	emailSender := NewMemoryEmailSender()
	srv := api.NewServer(store, cfg, api.WithEmailSender(emailSender))

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
