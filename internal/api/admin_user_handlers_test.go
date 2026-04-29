package api_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

// TestAdminCreateUser exercises POST /api/v1/admin/users:
//   - 401 without admin key
//   - 400 on invalid body
//   - 400 on missing email
//   - 201 on valid email+password + returned shape
//   - 409 on duplicate email
func TestAdminCreateUser(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	_ = ctx

	// 1. No admin key â†’ 401
	req, _ := http.NewRequest(http.MethodPost, ts.URL("/api/v1/admin/users"),
		bytes.NewReader([]byte(`{"email":"noauth@example.com"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no admin key: expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Invalid JSON body â†’ 400
	req2, _ := http.NewRequest(http.MethodPost, ts.URL("/api/v1/admin/users"),
		bytes.NewReader([]byte(`not json`)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+ts.AdminKey)
	resp, err = http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid body: expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. Missing email â†’ 400
	resp = ts.PostJSONWithAdminKey("/api/v1/admin/users", map[string]any{"name": "no email"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing email: expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 4. Valid create with password â†’ 201
	createEmail := "created-by-admin@example.com"
	resp = ts.PostJSONWithAdminKey("/api/v1/admin/users", map[string]any{
		"email":    createEmail,
		"password": "SuperSecret123!",
		"name":     "Created By Admin",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("valid create: expected 201, got %d", resp.StatusCode)
	}
	var body struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	ts.DecodeJSON(resp, &body)
	if body.ID == "" {
		t.Fatalf("response missing id")
	}
	if body.Email != createEmail {
		t.Fatalf("response email mismatch: want %q got %q", createEmail, body.Email)
	}

	// 5. Duplicate email â†’ 409
	resp = ts.PostJSONWithAdminKey("/api/v1/admin/users", map[string]any{
		"email":    createEmail,
		"password": "SuperSecret123!",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate: expected 409, got %d", resp.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	ts.DecodeJSON(resp, &errBody)
	if errBody.Error != "email_exists" {
		t.Fatalf("duplicate: expected error=email_exists, got %q", errBody.Error)
	}
}
