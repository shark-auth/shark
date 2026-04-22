package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// mountDCRRotationRouter sets up a chi router with all DCR endpoints including
// the new rotation endpoints.
func mountDCRRotationRouter(srv *Server) chi.Router {
	r := chi.NewRouter()
	r.Post("/oauth/register", srv.HandleDCRRegister)
	r.Get("/oauth/register/{client_id}", srv.HandleDCRGet)
	r.Put("/oauth/register/{client_id}", srv.HandleDCRUpdate)
	r.Delete("/oauth/register/{client_id}", srv.HandleDCRDelete)
	r.Post("/oauth/register/{client_id}/secret", srv.HandleDCRRotateSecret)
	r.Delete("/oauth/register/{client_id}/registration-token", srv.HandleDCRRotateRegistrationToken)
	return r
}

// registerClient is a test helper that registers a DCR client and returns the response body.
func registerClient(t *testing.T, ts *httptest.Server) map[string]interface{} {
	t.Helper()
	payload := map[string]interface{}{
		"client_name": "Rotation Test Client",
		"grant_types": []string{"client_credentials"},
	}
	resp := postJSON(t, ts, "/oauth/register", payload, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("register: expected 201, got %d: %s", resp.StatusCode, body)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding register response: %v", err)
	}
	return result
}

// TestDCR_RotateSecret_HappyPath verifies that POST /oauth/register/{id}/secret
// returns a new secret and that the new secret differs from the original.
func TestDCR_RotateSecret_HappyPath(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRotationRouter(srv))
	defer ts.Close()

	reg := registerClient(t, ts)
	clientID := reg["client_id"].(string)
	origSecret := reg["client_secret"].(string)
	regToken := reg["registration_access_token"].(string)

	// Rotate the secret.
	resp := postJSON(t, ts, "/oauth/register/"+clientID+"/secret", nil, regToken)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("rotate: expected 200, got %d: %s", resp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck

	newSecret, ok := result["client_secret"].(string)
	if !ok || newSecret == "" {
		t.Fatal("rotate: expected client_secret in response")
	}
	if newSecret == origSecret {
		t.Error("rotate: new secret must differ from original")
	}
	if result["client_id"] != clientID {
		t.Errorf("rotate: expected client_id %q, got %q", clientID, result["client_id"])
	}

	// The DB should now have old_secret_hash set.
	agent, err := store.GetAgentByClientID(context.Background(), clientID)
	if err != nil {
		t.Fatalf("fetching agent: %v", err)
	}
	if agent.OldSecretHash == "" {
		t.Error("rotate: expected old_secret_hash to be set in DB")
	}
	if agent.OldSecretExpiresAt == nil {
		t.Error("rotate: expected old_secret_expires_at to be set in DB")
	}
	// Grace window should be ~1 hour in the future.
	remaining := time.Until(*agent.OldSecretExpiresAt)
	if remaining < 50*time.Minute || remaining > 70*time.Minute {
		t.Errorf("rotate: expected ~1h grace window, got %v", remaining)
	}

	// New client_secret_hash must match SHA-256 of newSecret.
	h := sha256.Sum256([]byte(newSecret))
	expectedHash := hex.EncodeToString(h[:])
	if agent.ClientSecretHash != expectedHash {
		t.Error("rotate: new secret hash mismatch")
	}

}

// TestDCR_RotateSecret_OldSecretWorksInGrace verifies that the old secret is
// still accepted by fosite's GetClient during the grace window.
func TestDCR_RotateSecret_OldSecretWorksInGrace(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRotationRouter(srv))
	defer ts.Close()

	reg := registerClient(t, ts)
	clientID := reg["client_id"].(string)
	origSecret := reg["client_secret"].(string)
	regToken := reg["registration_access_token"].(string)

	// Rotate — old secret is still in grace window.
	rotResp := postJSON(t, ts, "/oauth/register/"+clientID+"/secret", nil, regToken)
	rotResp.Body.Close()

	// Fetch the fosite client via GetClient. During grace window, RotatedSecrets
	// should contain the old hash.
	client, err := srv.Store.GetClient(context.Background(), clientID)
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}

	// Verify old secret still accepted via hasher.
	hasher := &SHA256Hasher{}
	if err := hasher.Compare(context.Background(), client.GetHashedSecret(), []byte(origSecret)); err != nil {
		// Try rotated secrets.
		rotClient, ok := client.(interface{ GetRotatedHashes() [][]byte })
		if !ok {
			t.Fatal("client does not implement GetRotatedHashes")
		}
		hashes := rotClient.GetRotatedHashes()
		if len(hashes) == 0 {
			t.Fatal("rotate: expected rotated hash in grace window, got none")
		}
		found := false
		for _, h := range hashes {
			if hasher.Compare(context.Background(), h, []byte(origSecret)) == nil {
				found = true
				break
			}
		}
		if !found {
			t.Error("rotate: old secret not accepted in grace window")
		}
	}

	_ = store
}

// TestDCR_RotateSecret_OldSecretRejectedAfterGrace verifies that the old secret
// is rejected once old_secret_expires_at is in the past.
func TestDCR_RotateSecret_OldSecretRejectedAfterGrace(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRotationRouter(srv))
	defer ts.Close()

	reg := registerClient(t, ts)
	clientID := reg["client_id"].(string)
	origSecret := reg["client_secret"].(string)
	regToken := reg["registration_access_token"].(string)

	// Rotate.
	rotResp := postJSON(t, ts, "/oauth/register/"+clientID+"/secret", nil, regToken)
	rotResp.Body.Close()

	// Manipulate DB: expire the old secret immediately.
	agent, err := store.GetAgentByClientID(context.Background(), clientID)
	if err != nil {
		t.Fatalf("fetching agent: %v", err)
	}
	pastTime := time.Now().UTC().Add(-2 * time.Hour) // 2 hours ago
	if err := store.RotateDCRClientSecret(
		context.Background(),
		agent.ID,
		agent.ClientSecretHash,
		agent.OldSecretHash,
		pastTime,
	); err != nil {
		t.Fatalf("expiring old secret: %v", err)
	}

	// Now GetClient should not expose the old hash as a rotated secret.
	client, err := srv.Store.GetClient(context.Background(), clientID)
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}

	hasher := &SHA256Hasher{}
	// Old secret must fail against current hash.
	if err := hasher.Compare(context.Background(), client.GetHashedSecret(), []byte(origSecret)); err == nil {
		t.Error("old secret accepted against current hash after expiry — should fail")
	}

	// Rotated secrets must be empty or not contain the old hash.
	type rotatedClient interface{ GetRotatedHashes() [][]byte }
	if rc, ok := client.(rotatedClient); ok {
		for _, h := range rc.GetRotatedHashes() {
			if hasher.Compare(context.Background(), h, []byte(origSecret)) == nil {
				t.Error("old secret still accepted in rotated hashes after expiry")
			}
		}
	}
}

// TestDCR_RotateRegistrationToken_HappyPath verifies that
// DELETE /oauth/register/{id}/registration-token issues a new token and the
// old token is immediately invalidated.
func TestDCR_RotateRegistrationToken_HappyPath(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRotationRouter(srv))
	defer ts.Close()

	reg := registerClient(t, ts)
	clientID := reg["client_id"].(string)
	origToken := reg["registration_access_token"].(string)

	// Rotate the registration token using DELETE.
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/oauth/register/"+clientID+"/registration-token", nil)
	req.Header.Set("Authorization", "Bearer "+origToken)
	rotResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("rotation request failed: %v", err)
	}
	defer rotResp.Body.Close()

	if rotResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(rotResp.Body)
		t.Fatalf("rotate-token: expected 200, got %d: %s", rotResp.StatusCode, body)
	}

	var result map[string]interface{}
	json.NewDecoder(rotResp.Body).Decode(&result) //nolint:errcheck

	newToken, ok := result["registration_access_token"].(string)
	if !ok || newToken == "" {
		t.Fatal("rotate-token: expected registration_access_token in response")
	}
	if newToken == origToken {
		t.Error("rotate-token: new token must differ from original")
	}

	// Old token must now be rejected.
	getResp := getWithBearer(t, ts, "/oauth/register/"+clientID, origToken)
	getResp.Body.Close()
	if getResp.StatusCode == http.StatusOK {
		t.Error("rotate-token: old token must be rejected after rotation")
	}

	// New token must be accepted.
	getResp2 := getWithBearer(t, ts, "/oauth/register/"+clientID, newToken)
	defer getResp2.Body.Close()
	if getResp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(getResp2.Body)
		t.Fatalf("rotate-token: new token rejected: %d: %s", getResp2.StatusCode, body)
	}

	// Verify new hash in DB matches SHA-256 of new token.
	dcr, err := store.GetDCRClient(context.Background(), clientID)
	if err != nil {
		t.Fatalf("fetching DCR client: %v", err)
	}
	h := sha256.Sum256([]byte(newToken))
	expectedHash := hex.EncodeToString(h[:])
	if dcr.RegistrationTokenHash != expectedHash {
		t.Error("rotate-token: DB hash mismatch")
	}
}

// TestDCR_RotateSecret_RequiresValidToken verifies that secret rotation requires
// a valid registration_access_token.
func TestDCR_RotateSecret_RequiresValidToken(t *testing.T) {
	srv, _ := newTestOAuthServer(t)
	ts := httptest.NewServer(mountDCRRotationRouter(srv))
	defer ts.Close()

	reg := registerClient(t, ts)
	clientID := reg["client_id"].(string)

	// Wrong token.
	resp := postJSON(t, ts, "/oauth/register/"+clientID+"/secret", nil, "wrongtoken")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, body)
	}
}
