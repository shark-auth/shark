package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/shark-auth/shark/internal/testutil"
)

// TestMFA_FreshEnroll verifies that a fresh user can enroll in MFA successfully.
func TestMFA_FreshEnroll(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ts.SignupAndVerify("mfa-fresh@example.com", "SecurePassword123", "Fresh User")

	resp := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("fresh enroll: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var result map[string]interface{}
	ts.DecodeJSON(resp, &result)

	if result["secret"] == "" || result["secret"] == nil {
		t.Error("fresh enroll: expected non-empty secret")
	}
	if result["qr_uri"] == "" || result["qr_uri"] == nil {
		t.Error("fresh enroll: expected non-empty qr_uri")
	}
}

// TestMFA_ReEnrollWhenPending verifies that a user can re-enroll when their
// prior enrollment was never verified (mfa_verified_at IS NULL).
func TestMFA_ReEnrollWhenPending(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ts.SignupAndVerify("mfa-reenroll@example.com", "SecurePassword123", "Re-Enroll User")

	// First enroll â€” pending (not yet verified).
	resp1 := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	if resp1.StatusCode != http.StatusOK {
		body := readBody(t, resp1)
		t.Fatalf("first enroll: expected 200, got %d: %s", resp1.StatusCode, body)
	}
	resp1.Body.Close()

	// Re-enroll without verifying â€” must succeed (replaces the pending secret).
	resp2 := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	if resp2.StatusCode != http.StatusOK {
		body := readBody(t, resp2)
		t.Fatalf("re-enroll pending: expected 200, got %d: %s", resp2.StatusCode, body)
	}
	var result2 map[string]interface{}
	ts.DecodeJSON(resp2, &result2)

	if result2["secret"] == "" || result2["secret"] == nil {
		t.Error("re-enroll: expected non-empty secret")
	}
}

// TestMFA_ReEnrollBlockedWhenVerified verifies that re-enroll is rejected when
// the user has already completed verification (mfa_verified_at IS NOT NULL).
func TestMFA_ReEnrollBlockedWhenVerified(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ts.SignupAndVerify("mfa-verified@example.com", "SecurePassword123", "Verified User")

	// Enroll.
	resp := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("enroll: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var enrollResult map[string]interface{}
	ts.DecodeJSON(resp, &enrollResult)
	secret := enrollResult["secret"].(string)

	// Verify to set mfa_verified_at.
	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("generating TOTP: %v", err)
	}
	verResp := ts.PostJSON("/api/v1/auth/mfa/verify", map[string]string{"code": code})
	if verResp.StatusCode != http.StatusOK {
		body := readBody(t, verResp)
		t.Fatalf("verify: expected 200, got %d: %s", verResp.StatusCode, body)
	}
	verResp.Body.Close()

	// Attempt re-enroll â€” must be blocked.
	resp2 := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		body := readBody(t, resp2)
		t.Fatalf("re-enroll after verify: expected 409, got %d: %s", resp2.StatusCode, body)
	}
	var errResult map[string]interface{}
	ts.DecodeJSON(resp2, &errResult)
	if errResult["error"] != "mfa_already_enabled" {
		t.Errorf("expected error=mfa_already_enabled, got %q", errResult["error"])
	}
}

// TestMFA_VerifySetsFlag verifies that a successful /mfa/verify sets
// mfa_verified_at in the database (F3.2).
func TestMFA_VerifySetsFlag(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ts.SignupAndVerify("mfa-flag@example.com", "SecurePassword123", "Flag User")

	// Enroll.
	resp := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("enroll: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var enrollResult map[string]interface{}
	ts.DecodeJSON(resp, &enrollResult)
	secret := enrollResult["secret"].(string)

	// Before verify â€” mfa_verified_at should be NULL.
	userBefore, err := ts.Store.GetUserByEmail(context.Background(), "mfa-flag@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail before verify: %v", err)
	}
	if userBefore.MFAVerifiedAt != nil {
		t.Error("before verify: mfa_verified_at should be NULL")
	}

	// Verify.
	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("generating TOTP: %v", err)
	}
	verResp := ts.PostJSON("/api/v1/auth/mfa/verify", map[string]string{"code": code})
	if verResp.StatusCode != http.StatusOK {
		body := readBody(t, verResp)
		t.Fatalf("verify: expected 200, got %d: %s", verResp.StatusCode, body)
	}
	verResp.Body.Close()

	// After verify â€” mfa_verified_at must be non-NULL.
	userAfter, err := ts.Store.GetUserByEmail(context.Background(), "mfa-flag@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail after verify: %v", err)
	}
	if userAfter.MFAVerifiedAt == nil {
		t.Error("after verify: expected mfa_verified_at to be set")
	}
	if *userAfter.MFAVerifiedAt == "" {
		t.Error("after verify: mfa_verified_at is empty string")
	}
}
