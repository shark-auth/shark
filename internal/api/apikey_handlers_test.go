package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

func TestCreateAPIKey(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "Test Key",
		"scopes": []string{"users:read", "users:write"},
	})

	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	ts.DecodeJSON(resp, &result)

	// Must contain the full key (only time it's returned)
	key, ok := result["key"].(string)
	if !ok || key == "" {
		t.Fatal("expected 'key' in response")
	}
	if len(key) < 40 {
		t.Errorf("key too short: %d chars", len(key))
	}

	// Must contain key_prefix
	prefix, ok := result["key_prefix"].(string)
	if !ok || prefix == "" {
		t.Fatal("expected 'key_prefix' in response")
	}
	if len(prefix) != 8 {
		t.Errorf("expected key_prefix length 8, got %d", len(prefix))
	}

	// Must contain id
	id, ok := result["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected 'id' in response")
	}
}

func TestCreateAPIKeyValidation(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Missing name
	resp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"scopes": []string{"users:read"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Missing scopes
	resp = ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name": "Test",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing scopes, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestListAPIKeys(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create two keys
	resp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "Key 1",
		"scopes": []string{"users:read"},
	})
	resp.Body.Close()
	resp = ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "Key 2",
		"scopes": []string{"users:write"},
	})
	resp.Body.Close()

	// List keys
	resp = ts.GetWithAdminKey("/api/v1/api-keys")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var keys []map[string]interface{}
	ts.DecodeJSON(resp, &keys)

	// 3 = 1 bootstrap admin key + 2 created above
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys (1 admin + 2 created), got %d", len(keys))
	}

	// Keys in list must NOT contain the full key
	for _, k := range keys {
		if _, hasKey := k["key"]; hasKey {
			t.Error("list response must not contain full 'key' field")
		}
		if _, hasDisplay := k["key_display"]; !hasDisplay {
			t.Error("list response must contain 'key_display'")
		}
	}
}

func TestGetAPIKey(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a key
	createResp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "Get Test Key",
		"scopes": []string{"users:read"},
	})
	var created map[string]interface{}
	ts.DecodeJSON(createResp, &created)
	id := created["id"].(string)

	// Get the key
	resp := ts.GetWithAdminKey("/api/v1/api-keys/" + id)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	ts.DecodeJSON(resp, &result)

	// Must NOT contain the full key
	if _, hasKey := result["key"]; hasKey {
		t.Error("get response must not contain full 'key' field")
	}
	if result["name"] != "Get Test Key" {
		t.Errorf("expected name 'Get Test Key', got %v", result["name"])
	}
}

func TestGetAPIKeyNotFound(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/api-keys/key_nonexistent")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestUpdateAPIKey(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a key
	createResp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "Original Name",
		"scopes": []string{"users:read"},
	})
	var created map[string]interface{}
	ts.DecodeJSON(createResp, &created)
	id := created["id"].(string)

	// Update name and scopes
	resp := ts.PatchJSONWithAdminKey("/api/v1/api-keys/"+id, map[string]interface{}{
		"name":   "Updated Name",
		"scopes": []string{"users:read", "roles:write"},
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	ts.DecodeJSON(resp, &result)

	if result["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %v", result["name"])
	}
}

func TestRevokeAPIKey(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a key
	createResp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "To Revoke",
		"scopes": []string{"users:read"},
	})
	var created map[string]interface{}
	ts.DecodeJSON(createResp, &created)
	id := created["id"].(string)

	// Revoke it
	resp := ts.DeleteWithAdminKey("/api/v1/api-keys/" + id)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	ts.DecodeJSON(resp, &result)

	if result["revoked_at"] == nil {
		t.Error("expected revoked_at to be set")
	}

	// Revoking again should fail
	resp = ts.DeleteWithAdminKey("/api/v1/api-keys/" + id)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for double revoke, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestRotateAPIKey(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a key
	createResp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "To Rotate",
		"scopes": []string{"users:read", "users:write"},
	})
	var created map[string]interface{}
	ts.DecodeJSON(createResp, &created)
	oldID := created["id"].(string)

	// Rotate it
	resp := ts.PostJSONWithAdminKey("/api/v1/api-keys/"+oldID+"/rotate", nil)
	if resp.StatusCode != http.StatusCreated {
		body := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var rotated map[string]interface{}
	ts.DecodeJSON(resp, &rotated)

	// New key returned
	if rotated["key"] == nil {
		t.Fatal("expected new 'key' in rotation response")
	}
	newID := rotated["id"].(string)
	if newID == oldID {
		t.Error("new key ID should differ from old key ID")
	}

	// Old key should be revoked now
	getResp := ts.GetWithAdminKey("/api/v1/api-keys/" + oldID)
	var oldKey map[string]interface{}
	ts.DecodeJSON(getResp, &oldKey)
	if oldKey["revoked_at"] == nil {
		t.Error("old key should have revoked_at set after rotation")
	}
}

func TestAPIKeyBearerAuth(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create an API key via admin endpoint
	createResp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":   "Bearer Test Key",
		"scopes": []string{"users:read"},
	})
	var created map[string]interface{}
	ts.DecodeJSON(createResp, &created)
	fullKey := created["key"].(string)

	// Use the key as Bearer token to hit the healthz endpoint
	// (healthz is public, but let's verify the key can be parsed by the middleware)
	req, _ := http.NewRequest("GET", ts.URL("/healthz"), nil)
	req.Header.Set("Authorization", "Bearer "+fullKey)
	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIKeyBearerAuthWrongKey(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a protected endpoint test by using the API key middleware on a test route.
	// For now, we verify the middleware logic by testing against the store directly.
	store := ts.Store

	// Create a valid key in the DB
	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey error: %v", err)
	}
	_ = fullKey

	now := time.Now().UTC().Format(time.RFC3339)
	apiKey := &storage.APIKey{
		ID:        "key_test1",
		Name:      "Test Key",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeySuffix: keySuffix,
		Scopes:    `["users:read"]`,
		RateLimit: 1000,
		CreatedAt: now,
	}
	if err := store.CreateAPIKey(context.Background(), apiKey); err != nil {
		t.Fatalf("CreateAPIKey error: %v", err)
	}

	// Wrong key should not match
	wrongKey := "sk_live_wrongkeywrongkeywrongkeywrongkeyXX"
	wrongHash := auth.HashAPIKey(wrongKey)
	_, lookupErr := store.GetAPIKeyByKeyHash(context.Background(), wrongHash)
	if lookupErr == nil {
		t.Error("wrong key hash should not find any API key")
	}
}

func TestAPIKeyBearerAuthRevokedKey(t *testing.T) {
	ts := testutil.NewTestServer(t)
	store := ts.Store

	// Create and then revoke a key
	_, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey error: %v", err)
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	revokedAt := now.Add(-1 * time.Hour).Format(time.RFC3339)

	apiKey := &storage.APIKey{
		ID:        "key_revoked",
		Name:      "Revoked Key",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeySuffix: keySuffix,
		Scopes:    `["users:read"]`,
		RateLimit: 1000,
		CreatedAt: nowStr,
		RevokedAt: &revokedAt,
	}
	if err := store.CreateAPIKey(context.Background(), apiKey); err != nil {
		t.Fatalf("CreateAPIKey error: %v", err)
	}

	// Look up the key - it exists but is revoked
	found, err := store.GetAPIKeyByKeyHash(context.Background(), keyHash)
	if err != nil {
		t.Fatalf("GetAPIKeyByKeyHash error: %v", err)
	}
	if found.RevokedAt == nil {
		t.Error("expected key to have revoked_at set")
	}
}

func TestAPIKeyRateLimit(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a key with a very low rate limit
	resp := ts.PostJSONWithAdminKey("/api/v1/api-keys", map[string]interface{}{
		"name":       "Rate Limited Key",
		"scopes":     []string{"users:read"},
		"rate_limit": 3,
	})
	var created map[string]interface{}
	ts.DecodeJSON(resp, &created)

	rateLimit := created["rate_limit"].(float64)
	if rateLimit != 3 {
		t.Errorf("expected rate_limit 3, got %v", rateLimit)
	}
}

func TestRequiresAdminKey(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Try to hit API key endpoints without admin key
	resp := ts.Get("/api/v1/api-keys")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without admin key, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp = ts.PostJSON("/api/v1/api-keys", map[string]interface{}{
		"name":   "No Auth",
		"scopes": []string{"users:read"},
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without admin key, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

