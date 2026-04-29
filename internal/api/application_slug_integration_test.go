// Package api_test â€” integration tests for application slug handling
// (auto-generation, explicit validation, conflict detection).
package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

// createAppPayload is a minimal POST /api/v1/admin/apps request body.
type createAppPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug,omitempty"`
}

// createAppResponse mirrors the fields we care about in the response.
type createAppResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

func TestCreateApp_AutoGeneratesSlug(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/apps", createAppPayload{
		Name: "My Test App",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var body createAppResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Slug != "my-test-app" {
		t.Errorf("auto-slug: got %q, want %q", body.Slug, "my-test-app")
	}
}

func TestCreateApp_ExplicitSlug(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/apps", createAppPayload{
		Name: "Whatever Name",
		Slug: "my-explicit-slug",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var body createAppResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Slug != "my-explicit-slug" {
		t.Errorf("explicit slug: got %q, want %q", body.Slug, "my-explicit-slug")
	}
}

func TestCreateApp_InvalidSlugReturns400(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/apps", createAppPayload{
		Name: "Bad Slug App",
		Slug: "-invalid-slug-",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "invalid_slug" {
		t.Errorf("error code: got %q, want %q", body["error"], "invalid_slug")
	}
}

func TestCreateApp_DuplicateSlugReturns409(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// First creation â€” must succeed.
	resp1 := ts.PostJSONWithAdminKey("/api/v1/admin/apps", createAppPayload{
		Name: "First App",
		Slug: "colliding-slug",
	})
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d", resp1.StatusCode)
	}

	// Second creation with the same slug â€” must conflict.
	resp2 := ts.PostJSONWithAdminKey("/api/v1/admin/apps", createAppPayload{
		Name: "Second App",
		Slug: "colliding-slug",
	})
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("second create (duplicate slug): expected 409, got %d", resp2.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "slug_conflict" {
		t.Errorf("error code: got %q, want %q", body["error"], "slug_conflict")
	}
}

func TestSlugReturnedInListAndGet(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create app with known slug.
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/apps", createAppPayload{
		Name: "Listed App",
		Slug: "listed-app-slug",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}

	// List â€” slug must appear.
	listResp := ts.GetWithAdminKey("/api/v1/admin/apps")
	defer listResp.Body.Close()
	var listBody struct {
		Data []createAppResponse `json:"data"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	found := false
	for _, a := range listBody.Data {
		if a.Slug == "listed-app-slug" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("slug %q not found in list response", "listed-app-slug")
	}
}
