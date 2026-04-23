package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestAdminListUsers_SnakeCaseShape ensures the /api/v1/users admin response
// uses snake_case JSON keys (email_verified, mfa_enabled, last_login_at, etc.)
// and NOT the camelCase variants that broke the dashboard.
func TestAdminListUsers_SnakeCaseShape(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()
	lla := now.Add(-5 * time.Minute).UTC().Format(time.RFC3339)

	u := &storage.User{
		ID:            "usr_snaketest",
		Email:         "snaketest@example.com",
		EmailVerified: true,
		MFAEnabled:    true,
		MFAVerified:   true,
		HashType:      "argon2id",
		Metadata:      "{}",
		CreatedAt:     now.Format(time.RFC3339),
		UpdatedAt:     now.Format(time.RFC3339),
	}
	if err := ts.Store.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	// Set last_login_at via UpdateUser (CreateUser does not insert the column).
	u.LastLoginAt = &lla
	if err := ts.Store.UpdateUser(ctx, u); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/users?search=snaketest@example.com")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Users []struct {
			ID            string  `json:"id"`
			EmailVerified bool    `json:"email_verified"`
			MFAEnabled    bool    `json:"mfa_enabled"`
			CreatedAt     string  `json:"created_at"`
			UpdatedAt     string  `json:"updated_at"`
			LastLoginAt   *string `json:"last_login_at"`
		} `json:"users"`
		Total int `json:"total"`
	}
	ts.DecodeJSON(resp, &body)

	if body.Total == 0 || len(body.Users) == 0 {
		t.Fatalf("expected at least 1 user in response, got total=%d len=%d", body.Total, len(body.Users))
	}

	// Find our seeded user — list may contain others from parallel tests in theory.
	var found *struct {
		ID            string  `json:"id"`
		EmailVerified bool    `json:"email_verified"`
		MFAEnabled    bool    `json:"mfa_enabled"`
		CreatedAt     string  `json:"created_at"`
		UpdatedAt     string  `json:"updated_at"`
		LastLoginAt   *string `json:"last_login_at"`
	}
	for i := range body.Users {
		if body.Users[i].ID == "usr_snaketest" {
			found = &body.Users[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("seeded user usr_snaketest not found in response")
	}

	if !found.EmailVerified {
		t.Errorf("email_verified: expected true, got false (camelCase mismatch?)")
	}
	if !found.MFAEnabled {
		t.Errorf("mfa_enabled: expected true, got false (camelCase mismatch?)")
	}
	if found.LastLoginAt == nil || *found.LastLoginAt == "" {
		t.Errorf("last_login_at: expected non-empty string, got nil/empty")
	}
	if found.CreatedAt == "" {
		t.Errorf("created_at: expected non-empty string, got empty (createdAt tag bug?)")
	}
}

// TestAdminGetUser_SnakeCaseShape verifies the single-user endpoint also
// emits snake_case keys.
func TestAdminGetUser_SnakeCaseShape(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	u := &storage.User{
		ID:            "usr_getsnake",
		Email:         "getsnake@example.com",
		EmailVerified: true,
		MFAEnabled:    false,
		HashType:      "argon2id",
		Metadata:      "{}",
		CreatedAt:     now.Format(time.RFC3339),
		UpdatedAt:     now.Format(time.RFC3339),
	}
	if err := ts.Store.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/users/usr_getsnake")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		ID            string `json:"id"`
		EmailVerified bool   `json:"email_verified"`
		MFAEnabled    bool   `json:"mfa_enabled"`
		CreatedAt     string `json:"created_at"`
	}
	ts.DecodeJSON(resp, &body)

	if body.ID != "usr_getsnake" {
		t.Errorf("id: expected usr_getsnake, got %q", body.ID)
	}
	if !body.EmailVerified {
		t.Errorf("email_verified: expected true, got false")
	}
	if body.CreatedAt == "" {
		t.Errorf("created_at: expected non-empty (createdAt tag bug?)")
	}
}
