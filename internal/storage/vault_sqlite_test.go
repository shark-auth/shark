package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// newTestVaultProvider returns a minimal VaultProvider for testing.
func newTestVaultProvider(t *testing.T, name string, active bool) *storage.VaultProvider {
	t.Helper()
	nid, err := gonanoid.New(21)
	if err != nil {
		t.Fatalf("generate nanoid: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	return &storage.VaultProvider{
		ID:              "vp_" + nid,
		Name:            name,
		DisplayName:     "Display " + name,
		AuthURL:         "https://example.com/oauth/authorize",
		TokenURL:        "https://example.com/oauth/token",
		ClientID:        "client-" + name,
		ClientSecretEnc: "enc::ZmFrZQ==",
		Scopes:          []string{"read", "write"},
		IconURL:         "https://example.com/icon.png",
		Active:          active,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// newTestVaultConnection builds a connection for a given provider/user.
func newTestVaultConnection(t *testing.T, providerID, userID string) *storage.VaultConnection {
	t.Helper()
	nid, err := gonanoid.New(21)
	if err != nil {
		t.Fatalf("generate nanoid: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(1 * time.Hour)
	return &storage.VaultConnection{
		ID:              "vc_" + nid,
		ProviderID:      providerID,
		UserID:          userID,
		AccessTokenEnc:  "enc::YWNjZXNz",
		RefreshTokenEnc: "enc::cmVmcmVzaA==",
		TokenType:       "Bearer",
		Scopes:          []string{"read", "write"},
		ExpiresAt:       &expires,
		Metadata:        map[string]any{"team": "eng"},
		NeedsReauth:     false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func TestVaultProviderCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "google_calendar", true)

	// Create
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("CreateVaultProvider: %v", err)
	}

	// GetByID
	got, err := store.GetVaultProviderByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetVaultProviderByID: %v", err)
	}
	if got.Name != p.Name {
		t.Errorf("Name: got %q, want %q", got.Name, p.Name)
	}
	if got.ClientSecretEnc != p.ClientSecretEnc {
		t.Errorf("ClientSecretEnc round-trip mismatch")
	}
	if len(got.Scopes) != 2 || got.Scopes[0] != "read" {
		t.Errorf("Scopes: got %v", got.Scopes)
	}
	if got.IconURL != p.IconURL {
		t.Errorf("IconURL: got %q, want %q", got.IconURL, p.IconURL)
	}
	if !got.Active {
		t.Error("expected active=true")
	}

	// GetByName
	byName, err := store.GetVaultProviderByName(ctx, p.Name)
	if err != nil {
		t.Fatalf("GetVaultProviderByName: %v", err)
	}
	if byName.ID != p.ID {
		t.Errorf("GetVaultProviderByName: got %q, want %q", byName.ID, p.ID)
	}

	// Update
	p.DisplayName = "Renamed"
	p.Scopes = []string{"admin"}
	p.Active = false
	if err := store.UpdateVaultProvider(ctx, p); err != nil {
		t.Fatalf("UpdateVaultProvider: %v", err)
	}
	upd, err := store.GetVaultProviderByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if upd.DisplayName != "Renamed" {
		t.Errorf("DisplayName after update: got %q", upd.DisplayName)
	}
	if len(upd.Scopes) != 1 || upd.Scopes[0] != "admin" {
		t.Errorf("Scopes after update: %v", upd.Scopes)
	}
	if upd.Active {
		t.Error("expected active=false after update")
	}

	// Delete
	if err := store.DeleteVaultProvider(ctx, p.ID); err != nil {
		t.Fatalf("DeleteVaultProvider: %v", err)
	}
	_, err = store.GetVaultProviderByID(ctx, p.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Get after delete: got %v, want sql.ErrNoRows", err)
	}
}

func TestVaultProviderListActiveFilter(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	a := newTestVaultProvider(t, "alpha", true)
	b := newTestVaultProvider(t, "beta", true)
	c := newTestVaultProvider(t, "gamma", false)
	for _, p := range []*storage.VaultProvider{a, b, c} {
		if err := store.CreateVaultProvider(ctx, p); err != nil {
			t.Fatalf("CreateVaultProvider %q: %v", p.Name, err)
		}
	}

	all, err := store.ListVaultProviders(ctx, false)
	if err != nil {
		t.Fatalf("ListVaultProviders all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 providers, got %d", len(all))
	}

	activeOnly, err := store.ListVaultProviders(ctx, true)
	if err != nil {
		t.Fatalf("ListVaultProviders active: %v", err)
	}
	if len(activeOnly) != 2 {
		t.Errorf("expected 2 active providers, got %d", len(activeOnly))
	}
	for _, p := range activeOnly {
		if !p.Active {
			t.Errorf("provider %q should be active", p.Name)
		}
	}
}

func TestVaultProviderNameUnique(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p1 := newTestVaultProvider(t, "slack", true)
	if err := store.CreateVaultProvider(ctx, p1); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	p2 := newTestVaultProvider(t, "slack", true) // same name, different id
	if err := store.CreateVaultProvider(ctx, p2); err == nil {
		t.Fatal("expected UNIQUE constraint violation on duplicate name, got nil")
	}
}

func TestVaultConnectionCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Seed provider + user (FK dependencies)
	p := newTestVaultProvider(t, "github", true)
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	u := testutil.CreateUser(t, store, "vault-user@example.com", nil)

	c := newTestVaultConnection(t, p.ID, u.ID)

	// Create
	if err := store.CreateVaultConnection(ctx, c); err != nil {
		t.Fatalf("CreateVaultConnection: %v", err)
	}

	// GetByID
	got, err := store.GetVaultConnectionByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetVaultConnectionByID: %v", err)
	}
	if got.AccessTokenEnc != c.AccessTokenEnc {
		t.Errorf("AccessTokenEnc round-trip mismatch")
	}
	if got.RefreshTokenEnc != c.RefreshTokenEnc {
		t.Errorf("RefreshTokenEnc round-trip mismatch")
	}
	if got.TokenType != "Bearer" {
		t.Errorf("TokenType: got %q", got.TokenType)
	}
	if len(got.Scopes) != 2 {
		t.Errorf("Scopes: got %v", got.Scopes)
	}
	if got.Metadata["team"] != "eng" {
		t.Errorf("Metadata[team]: got %v", got.Metadata["team"])
	}
	if got.ExpiresAt == nil {
		t.Error("expected non-nil ExpiresAt")
	}
	if got.NeedsReauth {
		t.Error("expected needs_reauth=false initially")
	}

	// Get by (provider_id, user_id) composite
	byPU, err := store.GetVaultConnection(ctx, p.ID, u.ID)
	if err != nil {
		t.Fatalf("GetVaultConnection: %v", err)
	}
	if byPU.ID != c.ID {
		t.Errorf("GetVaultConnection: got id %q, want %q", byPU.ID, c.ID)
	}

	// List by user
	byUser, err := store.ListVaultConnectionsByUserID(ctx, u.ID)
	if err != nil {
		t.Fatalf("ListVaultConnectionsByUserID: %v", err)
	}
	if len(byUser) != 1 {
		t.Errorf("expected 1 connection for user, got %d", len(byUser))
	}

	// List by provider
	byProv, err := store.ListVaultConnectionsByProviderID(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListVaultConnectionsByProviderID: %v", err)
	}
	if len(byProv) != 1 {
		t.Errorf("expected 1 connection for provider, got %d", len(byProv))
	}

	// Full UpdateVaultConnection
	c.TokenType = "Bearer"
	c.Scopes = []string{"admin"}
	c.Metadata = map[string]any{"team": "security"}
	if err := store.UpdateVaultConnection(ctx, c); err != nil {
		t.Fatalf("UpdateVaultConnection: %v", err)
	}
	upd, err := store.GetVaultConnectionByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if len(upd.Scopes) != 1 || upd.Scopes[0] != "admin" {
		t.Errorf("Scopes after update: %v", upd.Scopes)
	}
	if upd.Metadata["team"] != "security" {
		t.Errorf("Metadata after update: %v", upd.Metadata)
	}

	// Delete
	if err := store.DeleteVaultConnection(ctx, c.ID); err != nil {
		t.Fatalf("DeleteVaultConnection: %v", err)
	}
	_, err = store.GetVaultConnectionByID(ctx, c.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Get after delete: got %v, want sql.ErrNoRows", err)
	}
}

func TestVaultConnectionUniqueProviderUser(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "notion", true)
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	u := testutil.CreateUser(t, store, "unique-pair@example.com", nil)

	c1 := newTestVaultConnection(t, p.ID, u.ID)
	if err := store.CreateVaultConnection(ctx, c1); err != nil {
		t.Fatalf("first connection: %v", err)
	}

	c2 := newTestVaultConnection(t, p.ID, u.ID) // different id, same (provider, user)
	if err := store.CreateVaultConnection(ctx, c2); err == nil {
		t.Fatal("expected UNIQUE(provider_id, user_id) violation, got nil")
	}
}

func TestVaultConnectionFKCascadeOnProviderDelete(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "figma", true)
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	u := testutil.CreateUser(t, store, "cascade-provider@example.com", nil)
	c := newTestVaultConnection(t, p.ID, u.ID)
	if err := store.CreateVaultConnection(ctx, c); err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	// Delete provider -> connection should cascade away.
	if err := store.DeleteVaultProvider(ctx, p.ID); err != nil {
		t.Fatalf("DeleteVaultProvider: %v", err)
	}
	_, err := store.GetVaultConnectionByID(ctx, c.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("connection should have cascaded: got %v, want sql.ErrNoRows", err)
	}
}

func TestVaultConnectionFKCascadeOnUserDelete(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "linear", true)
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	u := testutil.CreateUser(t, store, "cascade-user@example.com", nil)
	c := newTestVaultConnection(t, p.ID, u.ID)
	if err := store.CreateVaultConnection(ctx, c); err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	// Delete user -> connection should cascade away.
	if err := store.DeleteUser(ctx, u.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err := store.GetVaultConnectionByID(ctx, c.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("connection should have cascaded on user delete: got %v, want sql.ErrNoRows", err)
	}
}

func TestVaultConnectionUpdateTokens(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "asana", true)
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	u := testutil.CreateUser(t, store, "tokens@example.com", nil)
	c := newTestVaultConnection(t, p.ID, u.ID)
	c.NeedsReauth = true // seed as needs_reauth — refresh should clear it
	if err := store.CreateVaultConnection(ctx, c); err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	newExpiry := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	if err := store.UpdateVaultConnectionTokens(
		ctx, c.ID,
		"enc::bmV3QWNjZXNz",
		"enc::bmV3UmVmcmVzaA==",
		&newExpiry,
	); err != nil {
		t.Fatalf("UpdateVaultConnectionTokens: %v", err)
	}

	got, err := store.GetVaultConnectionByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get after token update: %v", err)
	}
	if got.AccessTokenEnc != "enc::bmV3QWNjZXNz" {
		t.Errorf("AccessTokenEnc: got %q", got.AccessTokenEnc)
	}
	if got.RefreshTokenEnc != "enc::bmV3UmVmcmVzaA==" {
		t.Errorf("RefreshTokenEnc: got %q", got.RefreshTokenEnc)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(newExpiry) {
		t.Errorf("ExpiresAt: got %v, want %v", got.ExpiresAt, newExpiry)
	}
	if got.LastRefreshedAt == nil {
		t.Error("expected LastRefreshedAt to be set after token update")
	}
	if got.NeedsReauth {
		t.Error("needs_reauth should be cleared after successful token refresh")
	}

	// Empty refresh token handled as NULL without error.
	if err := store.UpdateVaultConnectionTokens(
		ctx, c.ID,
		"enc::YW5vdGhlcg==",
		"",
		nil,
	); err != nil {
		t.Fatalf("UpdateVaultConnectionTokens with empty refresh: %v", err)
	}
	got, err = store.GetVaultConnectionByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get after second update: %v", err)
	}
	if got.RefreshTokenEnc != "" {
		t.Errorf("empty refresh should remain empty, got %q", got.RefreshTokenEnc)
	}
	if got.ExpiresAt != nil {
		t.Errorf("nil ExpiresAt should clear column, got %v", got.ExpiresAt)
	}
}

func TestVaultConnectionMarkNeedsReauth(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "intercom", true)
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	u := testutil.CreateUser(t, store, "reauth@example.com", nil)
	c := newTestVaultConnection(t, p.ID, u.ID)
	if err := store.CreateVaultConnection(ctx, c); err != nil {
		t.Fatalf("seed connection: %v", err)
	}

	// Flip to true
	if err := store.MarkVaultConnectionNeedsReauth(ctx, c.ID, true); err != nil {
		t.Fatalf("MarkVaultConnectionNeedsReauth(true): %v", err)
	}
	got, err := store.GetVaultConnectionByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.NeedsReauth {
		t.Error("expected needs_reauth=true")
	}

	// Flip back to false
	if err := store.MarkVaultConnectionNeedsReauth(ctx, c.ID, false); err != nil {
		t.Fatalf("MarkVaultConnectionNeedsReauth(false): %v", err)
	}
	got, err = store.GetVaultConnectionByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.NeedsReauth {
		t.Error("expected needs_reauth=false after reset")
	}
}

func TestVaultProviderExtraAuthParamsRoundTrip(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "linear_custom", true)
	p.ExtraAuthParams = map[string]string{"prompt": "consent"}

	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("CreateVaultProvider: %v", err)
	}

	// Read back via GetByID.
	got, err := store.GetVaultProviderByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetVaultProviderByID: %v", err)
	}
	if got.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("ExtraAuthParams round-trip via GetByID: got %v, want prompt=consent", got.ExtraAuthParams)
	}

	// Read back via GetByName.
	byName, err := store.GetVaultProviderByName(ctx, p.Name)
	if err != nil {
		t.Fatalf("GetVaultProviderByName: %v", err)
	}
	if byName.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("ExtraAuthParams round-trip via GetByName: got %v", byName.ExtraAuthParams)
	}

	// Update: replace extra params.
	p.ExtraAuthParams = map[string]string{"audience": "api.example.com", "prompt": "consent"}
	if err := store.UpdateVaultProvider(ctx, p); err != nil {
		t.Fatalf("UpdateVaultProvider: %v", err)
	}
	upd, err := store.GetVaultProviderByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if upd.ExtraAuthParams["audience"] != "api.example.com" {
		t.Errorf("ExtraAuthParams after update: got %v", upd.ExtraAuthParams)
	}
	if upd.ExtraAuthParams["prompt"] != "consent" {
		t.Errorf("ExtraAuthParams[prompt] after update: got %v", upd.ExtraAuthParams)
	}

	// Provider with nil/empty extra params serialises as "{}" (no error).
	p2 := newTestVaultProvider(t, "no_extra_provider", false)
	p2.ExtraAuthParams = nil
	if err := store.CreateVaultProvider(ctx, p2); err != nil {
		t.Fatalf("CreateVaultProvider with nil ExtraAuthParams: %v", err)
	}
	got2, err := store.GetVaultProviderByID(ctx, p2.ID)
	if err != nil {
		t.Fatalf("GetVaultProviderByID nil extra: %v", err)
	}
	if got2.ExtraAuthParams == nil {
		t.Error("ExtraAuthParams should be non-nil empty map when stored as nil")
	}
}

func TestVaultConnectionEmptyOptionalFields(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	p := newTestVaultProvider(t, "mailchimp", true)
	if err := store.CreateVaultProvider(ctx, p); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	u := testutil.CreateUser(t, store, "minimal@example.com", nil)

	// Connection without refresh token, expires_at, or metadata.
	nid, _ := gonanoid.New(21)
	now := time.Now().UTC().Truncate(time.Second)
	c := &storage.VaultConnection{
		ID:             "vc_" + nid,
		ProviderID:     p.ID,
		UserID:         u.ID,
		AccessTokenEnc: "enc::bWluaW1hbA==",
		TokenType:      "Bearer",
		Scopes:         nil,
		Metadata:       nil,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := store.CreateVaultConnection(ctx, c); err != nil {
		t.Fatalf("CreateVaultConnection minimal: %v", err)
	}

	got, err := store.GetVaultConnectionByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("Get minimal: %v", err)
	}
	if got.RefreshTokenEnc != "" {
		t.Errorf("RefreshTokenEnc should be empty, got %q", got.RefreshTokenEnc)
	}
	if got.ExpiresAt != nil {
		t.Errorf("ExpiresAt should be nil, got %v", got.ExpiresAt)
	}
	if got.LastRefreshedAt != nil {
		t.Errorf("LastRefreshedAt should be nil, got %v", got.LastRefreshedAt)
	}
	if got.Scopes == nil {
		t.Error("Scopes should be non-nil empty slice")
	}
	if got.Metadata == nil {
		t.Error("Metadata should be non-nil empty map")
	}
}
