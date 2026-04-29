package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/testutil"
)

func newMFAManager(t *testing.T) (*auth.MFAManager, *testutil.TestServer) {
	t.Helper()
	ts := testutil.NewTestServer(t)
	m := auth.NewMFAManager(ts.Store, config.MFAConfig{
		Issuer:        "SharkAuth Test",
		RecoveryCodes: 10,
	})
	return m, ts
}

// createTestUser creates a user in the store and returns its ID.
func createTestUser(t *testing.T, ts *testutil.TestServer) string {
	t.Helper()
	resp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email":    "mfa-test@example.com",
		"password": "SecurePassword123",
		"name":     "MFA Test User",
	})
	if resp.StatusCode != 201 {
		t.Fatalf("failed to create test user: status %d", resp.StatusCode)
	}
	var result map[string]interface{}
	ts.DecodeJSON(resp, &result)
	return result["id"].(string)
}

func TestTOTPEnrollAndVerify(t *testing.T) {
	m, _ := newMFAManager(t)

	// Generate a TOTP secret
	secret, qrURI, err := m.GenerateSecret("test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if qrURI == "" {
		t.Fatal("expected non-empty QR URI")
	}

	// Verify QR URI contains expected components
	if !contains(qrURI, "otpauth://totp/") {
		t.Fatalf("QR URI missing otpauth prefix: %s", qrURI)
	}
	if !contains(qrURI, "secret="+secret) {
		t.Fatalf("QR URI missing secret: %s", qrURI)
	}

	// Generate a valid code at the current time
	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	// Validate should pass
	if !m.ValidateTOTP(secret, code) {
		t.Fatal("expected TOTP validation to pass for current code")
	}
}

func TestTOTPRejectsOldCode(t *testing.T) {
	m, _ := newMFAManager(t)

	secret, _, err := m.GenerateSecret("old-code@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}

	// Generate a code from 5 minutes ago (10 steps back at 30s each)
	oldTime := time.Now().UTC().Add(-5 * time.Minute)
	oldCode, err := totp.GenerateCodeCustom(secret, oldTime, totp.ValidateOpts{
		Period:    30,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	// Validate should fail â€” code is too old (skew=1 allows +/- 1 step = 30s)
	if m.ValidateTOTP(secret, oldCode) {
		t.Fatal("expected TOTP validation to reject 5-minute-old code")
	}
}

func TestRecoveryCodeOneTimeUse(t *testing.T) {
	m, ts := newMFAManager(t)
	ctx := context.Background()
	userID := createTestUser(t, ts)

	// Generate recovery codes
	codes, err := m.GenerateRecoveryCodes(ctx, userID)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes failed: %v", err)
	}
	if len(codes) != 10 {
		t.Fatalf("expected 10 codes, got %d", len(codes))
	}

	// Use the first code â€” should succeed
	ok, err := m.VerifyRecoveryCode(ctx, userID, codes[0])
	if err != nil {
		t.Fatalf("VerifyRecoveryCode failed: %v", err)
	}
	if !ok {
		t.Fatal("expected first use of recovery code to succeed")
	}

	// Use the same code again â€” should fail (one-time use)
	ok, err = m.VerifyRecoveryCode(ctx, userID, codes[0])
	if err != nil {
		t.Fatalf("VerifyRecoveryCode failed on second use: %v", err)
	}
	if ok {
		t.Fatal("expected second use of recovery code to fail")
	}
}

func TestRecoveryCodeUniqueness(t *testing.T) {
	m, ts := newMFAManager(t)
	ctx := context.Background()
	userID := createTestUser(t, ts)

	codes, err := m.GenerateRecoveryCodes(ctx, userID)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes failed: %v", err)
	}

	// All 10 codes must be distinct
	seen := make(map[string]bool)
	for i, code := range codes {
		if seen[code] {
			t.Fatalf("duplicate recovery code at index %d: %s", i, code)
		}
		seen[code] = true
	}
}

// contains checks if s contains substr (simple helper to avoid importing strings).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
