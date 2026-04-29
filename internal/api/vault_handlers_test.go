package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
	"github.com/shark-auth/shark/internal/vault"
)

// Minimal shape we care about in admin API responses for providers.
type testVaultProviderResp struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	DisplayName     string   `json:"display_name"`
	AuthURL         string   `json:"auth_url"`
	TokenURL        string   `json:"token_url"`
	ClientID        string   `json:"client_id"`
	ClientSecret    string   `json:"client_secret,omitempty"`
	ClientSecretEnc string   `json:"client_secret_enc,omitempty"`
	Scopes          []string `json:"scopes"`
	IconURL         string   `json:"icon_url,omitempty"`
	Active          bool     `json:"active"`
}

// seedVaultProvider creates a VaultProvider directly through the store so
// tests don't have to go through the admin API just to get a row present.
func seedVaultProvider(t *testing.T, ts *testutil.TestServer, name, displayName string) *storage.VaultProvider {
	t.Helper()
	p := &storage.VaultProvider{
		Name:        name,
		DisplayName: displayName,
		AuthURL:     "https://auth.example.com/authorize",
		TokenURL:    "https://auth.example.com/token",
		ClientID:    "test-client-id",
		Scopes:      []string{"read"},
		Active:      true,
	}
	m := vault.NewManager(ts.Store, ts.APIServer.FieldEncryptor)
	if err := m.CreateProvider(context.Background(), p, "test-secret"); err != nil {
		t.Fatalf("seed vault provider: %v", err)
	}
	return p
}

func TestCreateVaultProvider_FromTemplate(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/vault/providers", map[string]any{
		"template":      "google_calendar",
		"client_id":     "gcal-client.apps.googleusercontent.com",
		"client_secret": "GOCSPX-abc123",
	})
	if resp.StatusCode != http.StatusCreated {
		b := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}
	var got testVaultProviderResp
	ts.DecodeJSON(resp, &got)

	if got.Name != "google_calendar" {
		t.Errorf("expected name=google_calendar, got %q", got.Name)
	}
	if got.DisplayName != "Google Calendar" {
		t.Errorf("expected display_name=Google Calendar, got %q", got.DisplayName)
	}
	if got.ClientID != "gcal-client.apps.googleusercontent.com" {
		t.Errorf("client_id mismatch: %q", got.ClientID)
	}
	if got.ClientSecret != "" || got.ClientSecretEnc != "" {
		t.Errorf("response must not leak secrets (secret=%q, enc=%q)", got.ClientSecret, got.ClientSecretEnc)
	}
	if got.AuthURL == "" || got.TokenURL == "" {
		t.Error("template URLs not applied")
	}
	if !strings.HasPrefix(got.ID, "vp_") {
		t.Errorf("expected vp_ prefix id, got %q", got.ID)
	}

	// Verify persisted with encrypted secret.
	row, err := ts.Store.GetVaultProviderByID(context.Background(), got.ID)
	if err != nil || row == nil {
		t.Fatalf("provider not persisted: %v", err)
	}
	if row.ClientSecretEnc == "" {
		t.Error("client_secret_enc must be set on stored row")
	}
	if row.ClientSecretEnc == "GOCSPX-abc123" {
		t.Error("client_secret must be encrypted, not stored plaintext")
	}
}

func TestCreateVaultProvider_Explicit(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/vault/providers", map[string]any{
		"name":          "custom_provider",
		"display_name":  "Custom Provider",
		"auth_url":      "https://custom.example.com/authorize",
		"token_url":     "https://custom.example.com/token",
		"client_id":     "custom-id",
		"client_secret": "custom-secret",
		"scopes":        []string{"scope.one", "scope.two"},
	})
	if resp.StatusCode != http.StatusCreated {
		b := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}
	var got testVaultProviderResp
	ts.DecodeJSON(resp, &got)
	if got.Name != "custom_provider" || got.DisplayName != "Custom Provider" {
		t.Errorf("fields not preserved: %+v", got)
	}
	if len(got.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %v", got.Scopes)
	}
	if got.ClientSecret != "" {
		t.Error("secret leaked in response")
	}
}

func TestCreateVaultProvider_DuplicateName(t *testing.T) {
	ts := testutil.NewTestServer(t)

	body := map[string]any{
		"template":      "slack",
		"client_id":     "slack-client",
		"client_secret": "slack-secret",
	}
	r1 := ts.PostJSONWithAdminKey("/api/v1/vault/providers", body)
	r1.Body.Close()
	if r1.StatusCode != http.StatusCreated {
		t.Fatalf("first create: %d", r1.StatusCode)
	}

	r2 := ts.PostJSONWithAdminKey("/api/v1/vault/providers", body)
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusConflict {
		b := readBody(t, r2)
		t.Fatalf("expected 409 on duplicate, got %d: %s", r2.StatusCode, b)
	}
	var err map[string]string
	ts.DecodeJSON(r2, &err)
	if err["error"] != "name_exists" {
		t.Errorf("expected error=name_exists, got %v", err)
	}
}

func TestCreateVaultProvider_MissingAuth(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// No admin key header â€” should 401.
	resp := ts.PostJSON("/api/v1/vault/providers", map[string]any{
		"template":      "slack",
		"client_id":     "x",
		"client_secret": "y",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without admin key, got %d", resp.StatusCode)
	}
}

func TestListVaultProviders(t *testing.T) {
	ts := testutil.NewTestServer(t)

	seedVaultProvider(t, ts, "p1", "Provider One")
	seedVaultProvider(t, ts, "p2", "Provider Two")

	resp := ts.GetWithAdminKey("/api/v1/vault/providers")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Data  []testVaultProviderResp `json:"data"`
		Total int                     `json:"total"`
	}
	ts.DecodeJSON(resp, &body)
	if body.Total != 2 || len(body.Data) != 2 {
		t.Fatalf("expected 2 providers, got total=%d len=%d", body.Total, len(body.Data))
	}
	// Ensure no ciphertext leaks into the list either.
	for _, p := range body.Data {
		if p.ClientSecret != "" || p.ClientSecretEnc != "" {
			t.Errorf("list leaked secret material: %+v", p)
		}
	}
}

func TestGetVaultProvider_NotFound(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/vault/providers/vp_doesnotexist")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateVaultProvider_RotatesSecret(t *testing.T) {
	ts := testutil.NewTestServer(t)
	p := seedVaultProvider(t, ts, "rotatable", "Rotatable")

	before, _ := ts.Store.GetVaultProviderByID(context.Background(), p.ID)
	if before == nil || before.ClientSecretEnc == "" {
		t.Fatal("pre-condition: provider secret not set")
	}
	oldCipher := before.ClientSecretEnc

	resp := ts.PatchJSONWithAdminKey("/api/v1/vault/providers/"+p.ID, map[string]any{
		"display_name":  "Rotatable Renamed",
		"client_secret": "new-secret-value",
	})
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}
	var got testVaultProviderResp
	ts.DecodeJSON(resp, &got)
	if got.DisplayName != "Rotatable Renamed" {
		t.Errorf("display_name not updated: %q", got.DisplayName)
	}
	if got.ClientSecret != "" || got.ClientSecretEnc != "" {
		t.Error("rotate response leaked secret")
	}

	after, _ := ts.Store.GetVaultProviderByID(context.Background(), p.ID)
	if after.ClientSecretEnc == oldCipher {
		t.Error("ciphertext did not change after rotation")
	}
	if after.ClientSecretEnc == "new-secret-value" {
		t.Error("new secret stored in plaintext")
	}
}

func TestDeleteVaultProvider(t *testing.T) {
	ts := testutil.NewTestServer(t)
	p := seedVaultProvider(t, ts, "doomed", "Doomed")

	resp := ts.DeleteWithAdminKey("/api/v1/vault/providers/" + p.ID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	row, err := ts.Store.GetVaultProviderByID(context.Background(), p.ID)
	if err == nil && row != nil {
		t.Error("expected provider to be gone after delete")
	}
}

func TestListVaultTemplates(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/vault/templates")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		Data []vault.ProviderTemplate `json:"data"`
	}
	ts.DecodeJSON(resp, &body)
	// We ship 9 built-in templates.
	if len(body.Data) != 9 {
		t.Errorf("expected 9 templates, got %d", len(body.Data))
	}
}

func TestVaultConnectStart_RedirectsToProvider(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "vaultuser@x.io")
	p := seedVaultProvider(t, ts, "google_calendar", "Google Calendar")

	resp := ts.Get("/api/v1/vault/connect/" + p.Name)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc == "" || !strings.HasPrefix(loc, p.AuthURL) {
		t.Errorf("unexpected Location: %q (want prefix %q)", loc, p.AuthURL)
	}
	if !strings.Contains(loc, "client_id=test-client-id") {
		t.Errorf("Location missing client_id: %s", loc)
	}
	if !strings.Contains(loc, "state=") {
		t.Errorf("Location missing state: %s", loc)
	}

	// Cookie should be set with state:providerID packing.
	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "shark_vault_state" {
			found = true
			if !strings.Contains(c.Value, ":"+p.ID) {
				t.Errorf("state cookie missing provider binding: %q", c.Value)
			}
		}
	}
	if !found {
		t.Error("shark_vault_state cookie not set")
	}
}

// TestVaultCallback_ExchangesAndStores is intentionally skipped at the HTTP
// level in this task â€” exercising it requires swapping the provider's token
// URL mid-flight (the vault package builds oauth2.Config straight from the
// provider row). T6's smoke suite covers this path end-to-end; here we stop
// at the state-validation boundary, which is the interesting handler logic.
func TestVaultCallback_ExchangesAndStores(t *testing.T) {
	t.Skip("covered by T6 smoke tests; provider token URL swap is non-trivial at the HTTP layer")
}

// vaultCallbackGet issues a GET to the vault callback with the given query
// string and an optional state cookie. The session cookie set by
// loginFreshUser is carried automatically by the client's jar.
func vaultCallbackGet(t *testing.T, ts *testutil.TestServer, providerName, query string, stateCookie *http.Cookie) *http.Response {
	t.Helper()
	u := ts.URL("/api/v1/vault/callback/" + providerName)
	if query != "" {
		u += "?" + query
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatalf("build callback request: %v", err)
	}
	if stateCookie != nil {
		req.AddCookie(stateCookie)
	}
	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	return resp
}

// TestVaultCallback_MissingState verifies the handler rejects a callback that
// arrives without the state cookie the connect step set.
func TestVaultCallback_MissingState(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "cbmiss@x.io")
	p := seedVaultProvider(t, ts, "google_calendar", "Google Calendar")

	resp := vaultCallbackGet(t, ts, p.Name, "state=abc&code=xyz", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b := readBody(t, resp)
		t.Fatalf("expected 400 without state cookie, got %d: %s", resp.StatusCode, b)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "invalid_state" {
		t.Errorf("expected error=invalid_state, got %+v", body)
	}
}

// TestVaultCallback_StateMismatch sends a cookie with state=A but a query
// state=B. The handler must reject with 400.
func TestVaultCallback_StateMismatch(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "cbmismatch@x.io")
	p := seedVaultProvider(t, ts, "slack", "Slack")

	cookie := &http.Cookie{
		Name:  "shark_vault_state",
		Value: "stateA:" + p.ID,
	}
	resp := vaultCallbackGet(t, ts, p.Name, "state=stateB&code=xyz", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b := readBody(t, resp)
		t.Fatalf("expected 400 on state mismatch, got %d: %s", resp.StatusCode, b)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "invalid_state" {
		t.Errorf("expected error=invalid_state, got %+v", body)
	}
}

// TestVaultCallback_CookieProviderMismatch packs state:providerA in the cookie
// but calls /vault/callback/providerB. The handler must refuse so a cookie
// grabbed from one flow can't be replayed against a different provider.
func TestVaultCallback_CookieProviderMismatch(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "cbprovider@x.io")
	providerA := seedVaultProvider(t, ts, "provider_a", "Provider A")
	providerB := seedVaultProvider(t, ts, "provider_b", "Provider B")

	// Cookie is bound to providerA but request hits providerB's callback.
	cookie := &http.Cookie{
		Name:  "shark_vault_state",
		Value: "sharedstate:" + providerA.ID,
	}
	resp := vaultCallbackGet(t, ts, providerB.Name, "state=sharedstate&code=xyz", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b := readBody(t, resp)
		t.Fatalf("expected 400 on provider mismatch, got %d: %s", resp.StatusCode, b)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "invalid_state" {
		t.Errorf("expected error=invalid_state, got %+v", body)
	}
}

// TestVaultCallback_MissingCode verifies that a well-formed state round-trip
// with no authorization code still rejects cleanly, rather than blindly
// calling into the token exchange with an empty code.
func TestVaultCallback_MissingCode(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "cbnocode@x.io")
	p := seedVaultProvider(t, ts, "github", "GitHub")

	cookie := &http.Cookie{
		Name:  "shark_vault_state",
		Value: "goodstate:" + p.ID,
	}
	resp := vaultCallbackGet(t, ts, p.Name, "state=goodstate", cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b := readBody(t, resp)
		t.Fatalf("expected 400 on missing code, got %d: %s", resp.StatusCode, b)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "missing_code" {
		t.Errorf("expected error=missing_code, got %+v", body)
	}
}

// TestVaultCallback_ProviderError simulates a provider sending the user back
// with ?error=access_denied (e.g. consent rejected). The handler must surface
// an oauth_error without touching state cookies or attempting an exchange.
func TestVaultCallback_ProviderError(t *testing.T) {
	ts := testutil.NewTestServer(t)
	_ = loginFreshUser(t, ts, "cberror@x.io")
	p := seedVaultProvider(t, ts, "linear", "Linear")

	// Error params take precedence over state validation â€” no cookie needed.
	resp := vaultCallbackGet(t, ts, p.Name,
		"error=access_denied&error_description=user+cancelled", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		b := readBody(t, resp)
		t.Fatalf("expected 400 on provider error, got %d: %s", resp.StatusCode, b)
	}
	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "oauth_error" {
		t.Errorf("expected error=oauth_error, got %+v", body)
	}
	if !strings.Contains(body["message"], "access_denied") {
		t.Errorf("expected message to include provider error code, got %q", body["message"])
	}
}

func TestVaultGetToken_RequiresBearer(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedVaultProvider(t, ts, "slack", "Slack")

	// No Authorization header â†’ 401.
	resp := ts.Get("/api/v1/vault/slack/token")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without bearer, got %d", resp.StatusCode)
	}
	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header on bearer challenge")
	}
}

func TestListVaultConnections_SessionOnly(t *testing.T) {
	ts := testutil.NewTestServer(t)
	userID := loginFreshUser(t, ts, "connections@x.io")
	p := seedVaultProvider(t, ts, "github", "GitHub")

	// Seed one connection directly via store so we don't depend on the full
	// OAuth round-trip here.
	now := time.Now().UTC()
	exp := now.Add(time.Hour)
	conn := &storage.VaultConnection{
		ID:              "vc_test000000000000000001",
		ProviderID:      p.ID,
		UserID:          userID,
		AccessTokenEnc:  "enc::deadbeef",
		RefreshTokenEnc: "enc::cafebabe",
		TokenType:       "Bearer",
		Scopes:          []string{"repo"},
		ExpiresAt:       &exp,
		Metadata:        map[string]any{},
		NeedsReauth:     false,
		LastRefreshedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := ts.Store.CreateVaultConnection(context.Background(), conn); err != nil {
		t.Fatalf("seeding connection: %v", err)
	}

	resp := ts.Get("/api/v1/vault/connections")
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}
	var body struct {
		Data []struct {
			ID                  string `json:"id"`
			ProviderName        string `json:"provider_name"`
			ProviderDisplayName string `json:"provider_display_name"`
			AccessTokenEnc      string `json:"access_token_enc,omitempty"`
			RefreshTokenEnc     string `json:"refresh_token_enc,omitempty"`
		} `json:"data"`
		Total int `json:"total"`
	}
	ts.DecodeJSON(resp, &body)
	if body.Total != 1 || len(body.Data) != 1 {
		t.Fatalf("expected 1 connection, got total=%d len=%d", body.Total, len(body.Data))
	}
	row := body.Data[0]
	if row.ProviderName != "github" || row.ProviderDisplayName != "GitHub" {
		t.Errorf("expected provider enrichment, got %+v", row)
	}
	if row.AccessTokenEnc != "" || row.RefreshTokenEnc != "" {
		t.Error("connection list leaked token material")
	}
}

func TestDeleteVaultConnection_IDORProtection(t *testing.T) {
	ts := testutil.NewTestServer(t)
	p := seedVaultProvider(t, ts, "linear", "Linear")

	// User A creates their connection directly.
	userA := ts.SignupAndVerify("ownerA@x.io", "Hunter2Hunter2", "")
	now := time.Now().UTC()
	connA := &storage.VaultConnection{
		ID:              "vc_idor_test_a",
		ProviderID:      p.ID,
		UserID:          userA,
		AccessTokenEnc:  "enc::a",
		RefreshTokenEnc: "enc::a",
		TokenType:       "Bearer",
		Scopes:          []string{"read"},
		Metadata:        map[string]any{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := ts.Store.CreateVaultConnection(context.Background(), connA); err != nil {
		t.Fatalf("seeding connA: %v", err)
	}

	// User B logs in (their session is now active via the cookie jar) and
	// tries to delete user A's connection.
	_ = loginFreshUser(t, ts, "attackerB@x.io")

	resp := ts.Delete("/api/v1/vault/connections/" + connA.ID)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("IDOR: expected 404, got %d", resp.StatusCode)
	}

	// Connection must still exist.
	row, err := ts.Store.GetVaultConnectionByID(context.Background(), connA.ID)
	if err != nil || row == nil {
		t.Error("connection should survive IDOR attempt")
	}
}

// Verify chi's route trie picks static prefixes over the wildcard {provider}
// token route. This is called out in the task brief as a sanity check.
func TestVaultRoutePrecedence_ProvidersBeatsWildcard(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// /vault/providers must hit the admin list handler (200), not the
	// /vault/{provider}/token bearer handler (401 bearer-challenge).
	resp := ts.GetWithAdminKey("/api/v1/vault/providers")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("/vault/providers should match admin list, got %d: %s", resp.StatusCode, b)
	}
	// If the wildcard had won, /vault/providers would have tried bearer auth
	// and returned 401 + WWW-Authenticate. Make sure that header isn't set.
	if resp.Header.Get("WWW-Authenticate") != "" {
		t.Error("unexpected WWW-Authenticate on /vault/providers â€” wildcard leaked")
	}

	// And /vault/templates too.
	r2 := ts.GetWithAdminKey("/api/v1/vault/templates")
	defer r2.Body.Close()
	if r2.StatusCode != http.StatusOK {
		t.Fatalf("/vault/templates should match admin templates, got %d", r2.StatusCode)
	}

	// Discard any body so we don't leak a response.
	_ = json.NewDecoder(resp.Body).Decode(&struct{}{})
}

// testVaultProviderRespWithExtra extends the response decoder to include extra_auth_params.
type testVaultProviderRespWithExtra struct {
	testVaultProviderResp
	ExtraAuthParams map[string]string `json:"extra_auth_params"`
}

func TestCreateVaultProvider_ExtraAuthParams_Manual(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/vault/providers", map[string]any{
		"name":              "custom_with_extra",
		"display_name":      "Custom With Extra",
		"auth_url":          "https://extra.example.com/authorize",
		"token_url":         "https://extra.example.com/token",
		"client_id":         "extra-id",
		"client_secret":     "extra-secret",
		"extra_auth_params": map[string]string{"prompt": "consent", "audience": "api.example.com"},
	})
	if resp.StatusCode != http.StatusCreated {
		b := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}

	var got testVaultProviderRespWithExtra
	ts.DecodeJSON(resp, &got)
	if got.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("extra_auth_params[prompt] in response: got %q, want consent", got.ExtraAuthParams["prompt"])
	}
	if got.ExtraAuthParams["audience"] != "api.example.com" {
		t.Errorf("extra_auth_params[audience] in response: got %q, want api.example.com", got.ExtraAuthParams["audience"])
	}

	// Verify persisted correctly.
	row, err := ts.Store.GetVaultProviderByID(context.Background(), got.ID)
	if err != nil || row == nil {
		t.Fatalf("provider not persisted: %v", err)
	}
	if row.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("persisted ExtraAuthParams[prompt]: got %q, want consent", row.ExtraAuthParams["prompt"])
	}
	if row.ExtraAuthParams["audience"] != "api.example.com" {
		t.Errorf("persisted ExtraAuthParams[audience]: got %q, want api.example.com", row.ExtraAuthParams["audience"])
	}
}

func TestCreateVaultProvider_ExtraAuthParams_Template(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Create a linear provider via template â€” should inherit prompt=consent from template.
	resp := ts.PostJSONWithAdminKey("/api/v1/vault/providers", map[string]any{
		"template":      "linear",
		"client_id":     "linear-client-id",
		"client_secret": "linear-client-secret",
	})
	if resp.StatusCode != http.StatusCreated {
		b := readBody(t, resp)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}

	var got testVaultProviderRespWithExtra
	ts.DecodeJSON(resp, &got)
	if got.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("template-created linear provider: extra_auth_params[prompt] = %q, want consent", got.ExtraAuthParams["prompt"])
	}

	// Persist confirmed.
	row, err := ts.Store.GetVaultProviderByID(context.Background(), got.ID)
	if err != nil || row == nil {
		t.Fatalf("provider not persisted: %v", err)
	}
	if row.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("persisted extra_auth_params[prompt] for linear: got %q, want consent", row.ExtraAuthParams["prompt"])
	}
}

func TestUpdateVaultProvider_ExtraAuthParams(t *testing.T) {
	ts := testutil.NewTestServer(t)

	p := seedVaultProvider(t, ts, "patchable_extra", "Patchable Extra")

	patchResp := ts.PatchJSONWithAdminKey("/api/v1/vault/providers/"+p.ID, map[string]any{
		"extra_auth_params": map[string]string{"prompt": "select_account"},
	})
	if patchResp.StatusCode != http.StatusOK {
		b := readBody(t, patchResp)
		t.Fatalf("PATCH extra_auth_params: got %d: %s", patchResp.StatusCode, b)
	}

	var got testVaultProviderRespWithExtra
	ts.DecodeJSON(patchResp, &got)
	if got.ExtraAuthParams["prompt"] != "select_account" {
		t.Errorf("PATCH response extra_auth_params[prompt]: got %q, want select_account", got.ExtraAuthParams["prompt"])
	}

	// Confirm persisted.
	row, err := ts.Store.GetVaultProviderByID(context.Background(), p.ID)
	if err != nil || row == nil {
		t.Fatalf("provider not found after PATCH: %v", err)
	}
	if row.ExtraAuthParams["prompt"] != "select_account" {
		t.Errorf("persisted after PATCH: got %q, want select_account", row.ExtraAuthParams["prompt"])
	}
}
