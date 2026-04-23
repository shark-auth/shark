// Package api — tests for handleAppSnippet (A8).
package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

type snippetResponse struct {
	Framework string `json:"framework"`
	Snippets  []struct {
		Label string `json:"label"`
		Lang  string `json:"lang"`
		Code  string `json:"code"`
	} `json:"snippets"`
}

// snippetSeedApp returns the seeded default app (see seedTestDefaultApp).
func snippetSeedApp(t *testing.T, ts *testutil.TestServer) (id, clientID string) {
	t.Helper()
	app, err := ts.Store.GetDefaultApplication(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultApplication: %v", err)
	}
	return app.ID, app.ClientID
}

func TestAppSnippet_React_Default(t *testing.T) {
	ts := testutil.NewTestServer(t)
	id, clientID := snippetSeedApp(t, ts)

	resp := ts.GetWithAdminKey("/api/v1/admin/apps/" + id + "/snippet")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var got snippetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Framework != "react" {
		t.Errorf("framework: got %q, want react", got.Framework)
	}
	if len(got.Snippets) != 3 {
		t.Fatalf("snippets: got %d, want 3", len(got.Snippets))
	}
	// Provider-setup snippet must substitute the real ClientID.
	foundProvider := false
	for _, s := range got.Snippets {
		if s.Label == "Provider setup" {
			foundProvider = true
			if !strings.Contains(s.Code, clientID) {
				t.Errorf("provider snippet missing client_id %q: %s", clientID, s.Code)
			}
		}
	}
	if !foundProvider {
		t.Fatalf("no Provider setup snippet in response")
	}
}

func TestAppSnippet_ExplicitReact(t *testing.T) {
	ts := testutil.NewTestServer(t)
	id, clientID := snippetSeedApp(t, ts)

	resp := ts.GetWithAdminKey("/api/v1/admin/apps/" + id + "/snippet?framework=react")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}

	var got snippetResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Framework != "react" {
		t.Errorf("framework: got %q, want react", got.Framework)
	}
	if len(got.Snippets) != 3 {
		t.Fatalf("snippets: got %d, want 3", len(got.Snippets))
	}
	// Still substitutes the real ClientID.
	body, _ := json.Marshal(got)
	if !strings.Contains(string(body), clientID) {
		t.Errorf("body missing client_id %q: %s", clientID, body)
	}
}

func TestAppSnippet_NotImplemented(t *testing.T) {
	ts := testutil.NewTestServer(t)
	id, _ := snippetSeedApp(t, ts)

	resp := ts.GetWithAdminKey("/api/v1/admin/apps/" + id + "/snippet?framework=vue")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("status: got %d, want 501", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "framework_not_supported" {
		t.Errorf("error: got %q, want framework_not_supported", body["error"])
	}
}

func TestAppSnippet_AppNotFound(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/admin/apps/app_nonexistent/snippet")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", resp.StatusCode)
	}
}
