package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// seedOAuthUser creates a minimal user for OAuth storage tests.
func seedOAuthUser(t *testing.T, store *storage.SQLiteStore, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	u := &storage.User{
		ID:        id,
		Email:     id + "@example.com",
		HashType:  "argon2id",
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("seed user %q: %v", id, err)
	}
}

// TestOAuthConsent_CRUD creates a consent, lists it, and revokes it — covering
// CreateOAuthConsent, ListConsentsByUserID, and RevokeOAuthConsent.
func TestOAuthConsent_CRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := "usr_consent_1"
	seedOAuthUser(t, store, userID)

	future := time.Now().UTC().Add(30 * 24 * time.Hour)
	consent := &storage.OAuthConsent{
		ID:        "consent_test_1",
		UserID:    userID,
		ClientID:  "client_consent_1",
		Scope:     "openid profile",
		GrantedAt: time.Now().UTC(),
		ExpiresAt: &future,
	}
	if err := store.CreateOAuthConsent(ctx, consent); err != nil {
		t.Fatalf("CreateOAuthConsent: %v", err)
	}

	// GetActiveConsent returns the consent.
	got, err := store.GetActiveConsent(ctx, userID, "client_consent_1")
	if err != nil {
		t.Fatalf("GetActiveConsent: %v", err)
	}
	if got == nil {
		t.Fatal("expected consent, got nil")
	}
	if got.Scope != "openid profile" {
		t.Errorf("expected scope 'openid profile', got %q", got.Scope)
	}
	if got.ExpiresAt == nil {
		t.Error("expected ExpiresAt populated")
	}

	// GetActiveConsent with unknown client returns nil/nil (no error).
	none, err := store.GetActiveConsent(ctx, userID, "other_client")
	if err != nil {
		t.Errorf("unexpected error for unknown client: %v", err)
	}
	if none != nil {
		t.Errorf("expected nil for unknown client, got %+v", none)
	}

	// ListConsentsByUserID returns the consent.
	consents, err := store.ListConsentsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ListConsentsByUserID: %v", err)
	}
	if len(consents) != 1 {
		t.Fatalf("expected 1 consent, got %d", len(consents))
	}
	if consents[0].ClientID != "client_consent_1" {
		t.Errorf("unexpected client_id: %q", consents[0].ClientID)
	}

	// Revoke the consent.
	if err := store.RevokeOAuthConsent(ctx, consent.ID); err != nil {
		t.Fatalf("RevokeOAuthConsent: %v", err)
	}

	// After revocation, it shouldn't show up as active.
	afterRevoke, err := store.GetActiveConsent(ctx, userID, "client_consent_1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if afterRevoke != nil {
		t.Errorf("expected no active consent after revoke, got %+v", afterRevoke)
	}

	// ListConsentsByUserID should no longer return the revoked consent.
	consentsAfter, err := store.ListConsentsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("ListConsentsByUserID after revoke: %v", err)
	}
	if len(consentsAfter) != 0 {
		t.Errorf("expected 0 consents after revoke, got %d", len(consentsAfter))
	}
}

// TestOAuthToken_ListAndDPoP creates tokens via CreateOAuthToken, lists them
// by agent ID, then updates the DPoP JKT — covering ListOAuthTokensByAgentID,
// UpdateOAuthTokenDPoPJKT, and scanOAuthTokenFromRows.
func TestOAuthToken_ListAndDPoP(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := "usr_tok_1"
	seedOAuthUser(t, store, userID)

	// Create 2 tokens for the same agent_id.
	now := time.Now().UTC()
	tokens := []*storage.OAuthToken{
		{
			ID:        "tok_list_1",
			JTI:       "jti-list-1",
			ClientID:  "client-tok-1",
			AgentID:   "agent_tok_1",
			UserID:    userID,
			TokenType: "access",
			TokenHash: "hash1",
			Scope:     "openid",
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		},
		{
			ID:        "tok_list_2",
			JTI:       "jti-list-2",
			ClientID:  "client-tok-1",
			AgentID:   "agent_tok_1",
			UserID:    userID,
			TokenType: "refresh",
			TokenHash: "hash2",
			Scope:     "openid",
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now,
		},
	}
	for _, tok := range tokens {
		if err := store.CreateOAuthToken(ctx, tok); err != nil {
			t.Fatalf("CreateOAuthToken %q: %v", tok.ID, err)
		}
	}

	// List them back via ListOAuthTokensByAgentID.
	listed, err := store.ListOAuthTokensByAgentID(ctx, "agent_tok_1", 10)
	if err != nil {
		t.Fatalf("ListOAuthTokensByAgentID: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(listed))
	}

	// Also exercise the default limit (limit=0) path.
	listedDef, err := store.ListOAuthTokensByAgentID(ctx, "agent_tok_1", 0)
	if err != nil {
		t.Fatalf("ListOAuthTokensByAgentID default: %v", err)
	}
	if len(listedDef) != 2 {
		t.Errorf("expected 2 tokens (default limit), got %d", len(listedDef))
	}

	// Update the DPoP JKT on one token.
	if err := store.UpdateOAuthTokenDPoPJKT(ctx, "tok_list_1", "dpop-jkt-xyz"); err != nil {
		t.Fatalf("UpdateOAuthTokenDPoPJKT: %v", err)
	}
	got, err := store.GetOAuthTokenByJTI(ctx, "jti-list-1")
	if err != nil {
		t.Fatalf("GetOAuthTokenByJTI after update: %v", err)
	}
	if got.DPoPJKT != "dpop-jkt-xyz" {
		t.Errorf("expected DPoPJKT 'dpop-jkt-xyz', got %q", got.DPoPJKT)
	}
}

// TestOAuthCleanup_Expired exercises DeleteExpired* helpers.
func TestOAuthCleanup_Expired(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := "usr_cleanup_1"
	seedOAuthUser(t, store, userID)

	past := time.Now().UTC().Add(-time.Hour)

	// Expired auth code.
	err := store.CreateAuthorizationCode(ctx, &storage.OAuthAuthorizationCode{
		CodeHash:  "expired_code_hash",
		ClientID:  "client-cleanup",
		UserID:    userID,
		ExpiresAt: past,
		CreatedAt: past,
	})
	if err != nil {
		t.Fatalf("create expired auth code: %v", err)
	}

	// Expired + revoked token (required for DeleteExpiredOAuthTokens).
	expiredTok := &storage.OAuthToken{
		ID:        "tok_expired_1",
		JTI:       "jti-expired-1",
		ClientID:  "client-cleanup",
		UserID:    userID,
		TokenType: "access",
		TokenHash: "hash_expired_1",
		Scope:     "openid",
		ExpiresAt: past,
		CreatedAt: past,
	}
	if err := store.CreateOAuthToken(ctx, expiredTok); err != nil {
		t.Fatalf("create expired token: %v", err)
	}
	// Revoke it so the cleanup predicate matches.
	if err := store.RevokeOAuthToken(ctx, expiredTok.ID); err != nil {
		t.Fatalf("revoke expired token: %v", err)
	}

	// Expired device code.
	err = store.CreateDeviceCode(ctx, &storage.OAuthDeviceCode{
		DeviceCodeHash: "expired_device_hash",
		UserCode:       "EXPIR-ED01",
		ClientID:       "client-cleanup",
		Status:         "pending",
		PollInterval:   5,
		ExpiresAt:      past,
		CreatedAt:      past,
	})
	if err != nil {
		t.Fatalf("create expired device code: %v", err)
	}

	// Delete expired rows.
	codesDeleted, err := store.DeleteExpiredAuthorizationCodes(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredAuthorizationCodes: %v", err)
	}
	if codesDeleted != 1 {
		t.Errorf("expected 1 deleted auth code, got %d", codesDeleted)
	}

	tokensDeleted, err := store.DeleteExpiredOAuthTokens(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredOAuthTokens: %v", err)
	}
	if tokensDeleted != 1 {
		t.Errorf("expected 1 deleted token, got %d", tokensDeleted)
	}

	devicesDeleted, err := store.DeleteExpiredDeviceCodes(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredDeviceCodes: %v", err)
	}
	if devicesDeleted != 1 {
		t.Errorf("expected 1 deleted device code, got %d", devicesDeleted)
	}
}
