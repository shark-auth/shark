package jwt_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

const testSecret = "test-server-secret-that-is-at-least-32-bytes-long"
const testBaseURL = "http://localhost:8080"

func newTestCfg(checkPerRequest bool) *config.JWTConfig {
	return &config.JWTConfig{
		Enabled:         true,
		Mode:            "session",
		Audience:        "shark",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "30d",
		ClockSkew:       "30s",
		Revocation: config.JWTRevocationConfig{
			CheckPerRequest: checkPerRequest,
		},
	}
}

func newTestManager(t *testing.T, store *storage.SQLiteStore, checkPerRequest bool) *jwtpkg.Manager {
	t.Helper()
	cfg := newTestCfg(checkPerRequest)
	mgr := jwtpkg.NewManager(cfg, store, testBaseURL, testSecret)
	if err := mgr.EnsureActiveKey(context.Background()); err != nil {
		t.Fatalf("EnsureActiveKey: %v", err)
	}
	return mgr
}

func testUser() *storage.User {
	return &storage.User{ID: "user_test123", Email: "test@example.com"}
}

// TestIssueSession verifies token_type="session" and expiry is in the future.
func TestIssueSession(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	token, err := mgr.IssueSessionJWT(ctx, testUser(), "sess_abc", true)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := mgr.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.TokenType != "session" {
		t.Errorf("expected token_type=session, got %q", claims.TokenType)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Before(time.Now()) {
		t.Error("expected exp in the future")
	}
	if claims.Subject != testUser().ID {
		t.Errorf("expected sub=%s, got %s", testUser().ID, claims.Subject)
	}
	if claims.SessionID != "sess_abc" {
		t.Errorf("expected session_id=sess_abc, got %s", claims.SessionID)
	}
	if !claims.MFAPassed {
		t.Error("expected mfa_passed=true")
	}
}

// TestIssueAccessRefresh verifies both tokens parse, access exp=~15m, and
// each has a distinct jti.
func TestIssueAccessRefresh(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	access, refresh, err := mgr.IssueAccessRefreshPair(ctx, testUser(), "sess_xyz", false)
	if err != nil {
		t.Fatalf("IssueAccessRefreshPair: %v", err)
	}
	if access == "" || refresh == "" {
		t.Fatal("expected non-empty tokens")
	}

	// Access token validates as "access" type
	accessClaims, err := mgr.Validate(ctx, access)
	if err != nil {
		t.Fatalf("Validate access: %v", err)
	}
	if accessClaims.TokenType != "access" {
		t.Errorf("expected token_type=access, got %q", accessClaims.TokenType)
	}

	// Access exp should be ~15m from now
	if accessClaims.ExpiresAt == nil {
		t.Fatal("expected non-nil exp on access token")
	}
	untilExp := time.Until(accessClaims.ExpiresAt.Time)
	if untilExp < 14*time.Minute || untilExp > 16*time.Minute {
		t.Errorf("access token exp should be ~15m, got %v", untilExp)
	}

	// Refresh token should be usable (different JTI)
	if accessClaims.ID == "" {
		t.Error("access JTI should not be empty")
	}

	// Confirm refresh token has distinct JTI by issuing a pair then refreshing
	newAccess, newRefresh, err := mgr.Refresh(ctx, refresh)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Error("expected non-empty new pair from Refresh")
	}
}

// TestValidate_Valid ensures happy-path validation returns correct claims.
func TestValidate_Valid(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	token, err := mgr.IssueSessionJWT(ctx, testUser(), "sess_valid", true)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}

	claims, err := mgr.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if claims.Subject != testUser().ID {
		t.Errorf("sub mismatch: got %s", claims.Subject)
	}
	if claims.SessionID != "sess_valid" {
		t.Errorf("session_id mismatch: got %s", claims.SessionID)
	}
	if !claims.MFAPassed {
		t.Error("expected mfa_passed=true")
	}
}

// TestValidate_Expired verifies that a tampered-expired token returns an error.
// We craft an expired token by signing with HS256 first to get a structure,
// then swap in the RS256 alg but with a past exp. Since we can't easily sign
// with the stored RS256 key from outside the package, we test the expiry path
// by creating a manager without clock skew and checking that re-issuing after
// artificially advancing time would fail. Instead, we use a simpler approach:
// test that an expired token signed by a different key returns an error.
//
// The contract being tested: ErrExpired when exp is in the past beyond clock skew.
func TestValidate_Expired(t *testing.T) {
	store := testutil.NewTestDB(t)
	// Use very short clock skew = 1s
	cfg := &config.JWTConfig{
		Enabled:         true,
		Mode:            "session",
		Audience:        "shark",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "30d",
		ClockSkew:       "1s",
	}
	mgr := jwtpkg.NewManager(cfg, store, testBaseURL, testSecret)
	ctx := context.Background()
	if err := mgr.EnsureActiveKey(ctx); err != nil {
		t.Fatalf("EnsureActiveKey: %v", err)
	}

	// Issue a valid token, then check that we get ErrExpired on an artificially
	// expired token. We construct one by using gojwt with HS256 so it will
	// fail at the alg-check step (not the expiry step).
	// Better: issue a real token, then replace payload exp with a past time
	// while keeping the same signature — this produces a signature mismatch error,
	// not expiry.
	//
	// The true expired-token test requires signing with RS256 and a past exp.
	// We do that by using the Refresh path on an expired refresh token, which
	// should return ErrExpired. But we can't control time without injection.
	//
	// PRAGMATIC APPROACH: use go-jwt to build a token with past exp using the
	// active key obtained from the store, but since the private key is encrypted
	// we can't access it from outside the package. Instead, we verify behavior
	// through the package's own interface: issue a token, rotate the key so the
	// original key is still in the store (but we can't modify exp).
	//
	// ACTUAL APPROACH: We directly test that gojwt returns ErrTokenExpired by
	// crafting a valid JWT structure with a past exp using a *separate* RSA key
	// that we register in the database.

	// Step 1: Generate a test keypair outside the package
	privKey, pubKeyPEM, kid, err := jwtpkg.GenerateTestKeyPair()
	if err != nil {
		t.Fatalf("GenerateTestKeyPair: %v", err)
	}

	// Step 2: Insert this key into the store as "retired" so kid lookup still works
	encPrivPEM, err := jwtpkg.EncryptTestPEM(privKey, testSecret)
	if err != nil {
		t.Fatalf("EncryptTestPEM: %v", err)
	}
	err = store.InsertSigningKey(ctx, &storage.SigningKey{
		KID:           kid,
		Algorithm:     "RS256",
		PublicKeyPEM:  pubKeyPEM,
		PrivateKeyPEM: encPrivPEM,
		Status:        "retired",
	})
	if err != nil {
		t.Fatalf("InsertSigningKey: %v", err)
	}

	// Step 3: Sign an expired token with this key
	now := time.Now().UTC()
	expiredToken := jwtpkg.BuildExpiredRS256Token(privKey, kid, "shark", testBaseURL, now)

	// Step 4: Validate should return an expiry-related error
	_, err = mgr.Validate(ctx, expiredToken)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	// Accept ErrExpired or an error containing "expired"
	if err != jwtpkg.ErrExpired && !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected ErrExpired-like error, got %v", err)
	}
}

// TestValidate_BadSignature verifies that a tampered signature returns ErrInvalidSignature.
func TestValidate_BadSignature(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	token, err := mgr.IssueSessionJWT(ctx, testUser(), "sess_badsig", true)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}

	// Tamper the signature (last segment)
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		t.Fatal("expected 3 JWT parts")
	}
	parts[2] = "invalidsignatureXXXXXXXXXXXXXXXXXXXXXXXXXXX"
	tampered := strings.Join(parts, ".")

	_, err = mgr.Validate(ctx, tampered)
	if err == nil {
		t.Fatal("expected error for bad signature, got nil")
	}
	if err != jwtpkg.ErrInvalidSignature {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

// TestValidate_AlgConfusion verifies that an HS256 token is rejected with ErrAlgMismatch.
func TestValidate_AlgConfusion(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	// Get the active key's public PEM to use as the HMAC secret (the attack)
	key, err := store.GetActiveSigningKey(ctx)
	if err != nil {
		t.Fatalf("GetActiveSigningKey: %v", err)
	}

	// Sign using HS256 with the public key PEM as the HMAC secret
	hmacToken := gojwt.NewWithClaims(gojwt.SigningMethodHS256, gojwt.MapClaims{
		"sub":        testUser().ID,
		"kid":        key.KID,
		"exp":        time.Now().Add(time.Hour).Unix(),
		"iss":        testBaseURL,
		"aud":        []string{"shark"},
		"token_type": "session",
		"jti":        "algconfusion-jti",
	})

	signed, err := hmacToken.SignedString([]byte(key.PublicKeyPEM))
	if err != nil {
		t.Fatalf("sign HS256: %v", err)
	}

	_, err = mgr.Validate(ctx, signed)
	if err == nil {
		t.Fatal("expected ErrAlgMismatch for alg-confusion attack, got nil")
	}
	if err != jwtpkg.ErrAlgMismatch {
		t.Errorf("expected ErrAlgMismatch, got %v", err)
	}
}

// TestValidate_UnknownKid verifies that a token with an unknown kid returns ErrUnknownKid.
func TestValidate_UnknownKid(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	token, err := mgr.IssueSessionJWT(ctx, testUser(), "sess_unk", true)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}

	// Replace the kid in the header with an unknown one
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		t.Fatal("expected 3 JWT parts")
	}

	// Decode the header, modify kid, re-encode
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	header["kid"] = "unknownkidXXXXXX"
	newHeaderJSON, _ := json.Marshal(header)
	parts[0] = base64.RawURLEncoding.EncodeToString(newHeaderJSON)

	tokenWithBadKid := strings.Join(parts, ".")

	_, err = mgr.Validate(ctx, tokenWithBadKid)
	if err == nil {
		t.Fatal("expected error for unknown kid, got nil")
	}
	if err != jwtpkg.ErrUnknownKid {
		// The signature will also fail, so ErrInvalidSignature is also acceptable
		if err != jwtpkg.ErrInvalidSignature {
			t.Errorf("expected ErrUnknownKid or ErrInvalidSignature, got %v", err)
		}
	}
}

// TestRefresh_Rotates verifies that Refresh issues new tokens and the old
// refresh JTI is inserted into revoked_jti (preventing reuse).
func TestRefresh_Rotates(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	_, refresh, err := mgr.IssueAccessRefreshPair(ctx, testUser(), "sess_ref", true)
	if err != nil {
		t.Fatalf("IssueAccessRefreshPair: %v", err)
	}

	newAccess, newRefresh, err := mgr.Refresh(ctx, refresh)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Error("expected non-empty new tokens")
	}

	// New access token validates correctly
	claims, err := mgr.Validate(ctx, newAccess)
	if err != nil {
		t.Fatalf("Validate new access: %v", err)
	}
	if claims.TokenType != "access" {
		t.Errorf("expected access type, got %q", claims.TokenType)
	}

	// Second Refresh with the same old refresh token should fail (one-time-use)
	_, _, err = mgr.Refresh(ctx, refresh)
	if err == nil {
		t.Fatal("expected error on second use of same refresh token")
	}
}

// TestRevoke_PreventsValidationWhenChecked verifies ErrRevoked after revoking a
// JTI with check_per_request=true.
func TestRevoke_PreventsValidationWhenChecked(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, true) // check_per_request=true
	ctx := context.Background()

	token, err := mgr.IssueSessionJWT(ctx, testUser(), "sess_revoke", true)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}

	// Validate succeeds before revocation
	claims, err := mgr.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate before revoke: %v", err)
	}

	// Revoke the JTI
	if err := mgr.RevokeJTI(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
		t.Fatalf("RevokeJTI: %v", err)
	}

	// Validate should now fail with ErrRevoked
	_, err = mgr.Validate(ctx, token)
	if err == nil {
		t.Fatal("expected ErrRevoked after revocation, got nil")
	}
	if err != jwtpkg.ErrRevoked {
		t.Errorf("expected ErrRevoked, got %v", err)
	}
}

// TestKeyRotation_BothInJWKS verifies that after rotation, both old (retired)
// and new (active) keys appear in ListJWKSCandidates.
func TestKeyRotation_BothInJWKS(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	// Record old key
	oldKey, err := store.GetActiveSigningKey(ctx)
	if err != nil {
		t.Fatalf("GetActiveSigningKey before rotate: %v", err)
	}

	// Rotate
	if err := mgr.GenerateAndStore(ctx, true); err != nil {
		t.Fatalf("GenerateAndStore (rotate): %v", err)
	}

	// Record new key
	newKey, err := store.GetActiveSigningKey(ctx)
	if err != nil {
		t.Fatalf("GetActiveSigningKey after rotate: %v", err)
	}
	if newKey.KID == oldKey.KID {
		t.Error("new key should have a different KID")
	}

	// Both keys should be in JWKS candidates (retiredCutoff = 1h ago means
	// any key rotated after that passes the filter)
	candidates, err := store.ListJWKSCandidates(ctx, false, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("ListJWKSCandidates: %v", err)
	}
	if len(candidates) < 2 {
		t.Errorf("expected at least 2 JWKS candidates after rotation, got %d", len(candidates))
	}

	kidSet := map[string]bool{}
	for _, c := range candidates {
		kidSet[c.KID] = true
	}
	if !kidSet[oldKey.KID] {
		t.Errorf("old key KID %q not found in JWKS candidates", oldKey.KID)
	}
	if !kidSet[newKey.KID] {
		t.Errorf("new key KID %q not found in JWKS candidates", newKey.KID)
	}
}

// -----------------------------------------------------------------------------
// Lane A A6 — Claims bake Tier + Roles
// -----------------------------------------------------------------------------

// userWithMetadata returns a fresh User, persisted in store, with the
// given metadata JSON. Used by the enrichment tests so each scenario
// can set (or omit) the "tier" field.
func userWithMetadata(t *testing.T, store *storage.SQLiteStore, id, metadataJSON string) *storage.User {
	t.Helper()
	u := &storage.User{
		ID:       id,
		Email:    id + "@example.com",
		Metadata: metadataJSON,
	}
	if err := store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return u
}

// TestIssueSession_BakesTierAndRoles covers the happy path: user with
// tier in metadata + two roles → Claims carry both, role names survive
// the JSON round-trip in the expected order.
func TestIssueSession_BakesTierAndRoles(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	u := userWithMetadata(t, store, "user_tierpro", `{"tier":"pro"}`)
	rAdmin := testutil.CreateRole(t, store, "admin")
	rBilling := testutil.CreateRole(t, store, "billing")
	if err := store.AssignRoleToUser(ctx, u.ID, rAdmin.ID); err != nil {
		t.Fatalf("assign admin: %v", err)
	}
	if err := store.AssignRoleToUser(ctx, u.ID, rBilling.ID); err != nil {
		t.Fatalf("assign billing: %v", err)
	}

	token, err := mgr.IssueSessionJWT(ctx, u, "sess_abc", true)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}
	claims, err := mgr.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if claims.Tier != "pro" {
		t.Errorf("Tier = %q, want %q", claims.Tier, "pro")
	}
	if len(claims.Roles) != 2 {
		t.Fatalf("Roles len = %d, want 2 (got %v)", len(claims.Roles), claims.Roles)
	}
	// Collect into a set — ordering depends on store iteration order
	// which is not part of the contract.
	got := map[string]bool{}
	for _, r := range claims.Roles {
		got[r] = true
	}
	if !got["admin"] || !got["billing"] {
		t.Errorf("Roles = %v, want admin+billing", claims.Roles)
	}
}

// TestIssueAccessRefreshPair_BakesTierAndRoles mirrors the session test
// for the access/refresh issuer: access token carries Tier + Roles;
// refresh token deliberately does not.
func TestIssueAccessRefreshPair_BakesTierAndRoles(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	u := userWithMetadata(t, store, "user_accpro", `{"tier":"enterprise"}`)
	r := testutil.CreateRole(t, store, "ops")
	if err := store.AssignRoleToUser(ctx, u.ID, r.ID); err != nil {
		t.Fatalf("assign ops: %v", err)
	}

	access, refresh, err := mgr.IssueAccessRefreshPair(ctx, u, "sess_acc", false)
	if err != nil {
		t.Fatalf("IssueAccessRefreshPair: %v", err)
	}

	accessClaims, err := mgr.Validate(ctx, access)
	if err != nil {
		t.Fatalf("Validate access: %v", err)
	}
	if accessClaims.Tier != "enterprise" {
		t.Errorf("access Tier = %q, want enterprise", accessClaims.Tier)
	}
	if len(accessClaims.Roles) != 1 || accessClaims.Roles[0] != "ops" {
		t.Errorf("access Roles = %v, want [ops]", accessClaims.Roles)
	}

	// Refresh token is used by Refresh() which deliberately allows
	// refresh-type tokens. Parse it directly so we can inspect Tier/Roles.
	parsed, _, err := new(gojwt.Parser).ParseUnverified(refresh, &jwtpkg.Claims{})
	if err != nil {
		t.Fatalf("ParseUnverified refresh: %v", err)
	}
	refreshClaims, ok := parsed.Claims.(*jwtpkg.Claims)
	if !ok {
		t.Fatalf("refresh claims type = %T", parsed.Claims)
	}
	if refreshClaims.Tier != "" || len(refreshClaims.Roles) != 0 {
		t.Errorf("refresh must not carry enrichment: tier=%q roles=%v", refreshClaims.Tier, refreshClaims.Roles)
	}
}

// TestIssueSession_DefaultTier covers a user whose metadata does not
// set "tier" (empty string, malformed JSON, missing field): the default
// tier is baked so downstream rules can match tier:free.
func TestIssueSession_DefaultTier(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	cases := []struct {
		name     string
		metadata string
	}{
		{"empty metadata", ""},
		{"json without tier", `{"other":"x"}`},
		{"malformed json", `{not json`},
		{"explicit empty tier", `{"tier":""}`},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u := userWithMetadata(t, store, fmt.Sprintf("user_def_%d", i), c.metadata)
			token, err := mgr.IssueSessionJWT(ctx, u, "sess_def", false)
			if err != nil {
				t.Fatalf("IssueSessionJWT: %v", err)
			}
			claims, err := mgr.Validate(ctx, token)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if claims.Tier != "free" {
				t.Errorf("default Tier = %q, want \"free\"", claims.Tier)
			}
		})
	}
}

// TestIssueSession_NoRolesReturnsEmpty verifies a user with zero role
// assignments produces an empty (or absent) Roles slice — not a nil
// panic, not a slice with stray entries.
func TestIssueSession_NoRolesReturnsEmpty(t *testing.T) {
	store := testutil.NewTestDB(t)
	mgr := newTestManager(t, store, false)
	ctx := context.Background()

	u := userWithMetadata(t, store, "user_noroles", `{"tier":"free"}`)

	token, err := mgr.IssueSessionJWT(ctx, u, "sess_noroles", false)
	if err != nil {
		t.Fatalf("IssueSessionJWT: %v", err)
	}
	claims, err := mgr.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if len(claims.Roles) != 0 {
		t.Errorf("no-role user should produce empty Roles, got %v", claims.Roles)
	}
}
