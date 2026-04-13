package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

func TestMFAEnrollChallengeFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// 1. Signup + verify email (MFA enroll requires verified email)
	ts.SignupAndVerify("mfa-flow@example.com", "SecurePassword123", "MFA Flow User")

	// 2. Enroll in MFA
	resp := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("enroll: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var enrollResult map[string]interface{}
	ts.DecodeJSON(resp, &enrollResult)
	secret, ok := enrollResult["secret"].(string)
	if !ok || secret == "" {
		t.Fatal("enroll: expected non-empty secret in response")
	}
	qrURI, ok := enrollResult["qr_uri"].(string)
	if !ok || qrURI == "" {
		t.Fatal("enroll: expected non-empty qr_uri in response")
	}

	// 3. Generate a valid TOTP code and verify setup
	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("generating TOTP code: %v", err)
	}

	resp = ts.PostJSON("/api/v1/auth/mfa/verify", map[string]string{
		"code": code,
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("verify: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var verifyResult map[string]interface{}
	ts.DecodeJSON(resp, &verifyResult)
	if verifyResult["mfa_enabled"] != true {
		t.Fatalf("verify: expected mfa_enabled=true, got %v", verifyResult["mfa_enabled"])
	}
	recoveryCodes, ok := verifyResult["recovery_codes"].([]interface{})
	if !ok || len(recoveryCodes) == 0 {
		t.Fatal("verify: expected recovery codes in response")
	}

	// 4. Logout
	resp = ts.PostJSON("/api/v1/auth/logout", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("logout: expected 200, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 5. Login — should get mfa_required
	resp = ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email":    "mfa-flow@example.com",
		"password": "SecurePassword123",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var loginResult map[string]interface{}
	ts.DecodeJSON(resp, &loginResult)
	if loginResult["mfaRequired"] != true {
		t.Fatalf("login: expected mfaRequired=true, got %v", loginResult["mfaRequired"])
	}

	// 6. GET /me should fail (mfa_passed=false, blocked by RequireMFA middleware)
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusForbidden {
		body := readBody(t, resp)
		t.Fatalf("/me before challenge: expected 403, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// 7. Challenge with valid TOTP code
	code, err = totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Digits:   otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("generating TOTP code for challenge: %v", err)
	}

	resp = ts.PostJSON("/api/v1/auth/mfa/challenge", map[string]string{
		"code": code,
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("challenge: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var challengeResult map[string]interface{}
	ts.DecodeJSON(resp, &challengeResult)
	if challengeResult["email"] != "mfa-flow@example.com" {
		t.Fatalf("challenge: expected email mfa-flow@example.com, got %v", challengeResult["email"])
	}

	// 8. GET /me should now work (mfa_passed=true)
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("/me after challenge: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var meResult map[string]interface{}
	ts.DecodeJSON(resp, &meResult)
	if meResult["email"] != "mfa-flow@example.com" {
		t.Fatalf("/me: expected email mfa-flow@example.com, got %v", meResult["email"])
	}
}

func TestMFARecoveryCodeFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Signup + verify email (MFA enroll requires verified email)
	ts.SignupAndVerify("mfa-recovery@example.com", "SecurePassword123", "")

	// Enroll
	resp := ts.PostJSON("/api/v1/auth/mfa/enroll", nil)
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("enroll: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var enrollResult map[string]interface{}
	ts.DecodeJSON(resp, &enrollResult)
	secret := enrollResult["secret"].(string)

	// Verify with valid code
	code, _ := totp.GenerateCodeCustom(secret, time.Now().UTC(), totp.ValidateOpts{
		Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	resp = ts.PostJSON("/api/v1/auth/mfa/verify", map[string]string{"code": code})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("verify: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var verifyResult map[string]interface{}
	ts.DecodeJSON(resp, &verifyResult)
	recoveryCodes := verifyResult["recovery_codes"].([]interface{})
	firstCode := recoveryCodes[0].(string)

	// Logout
	ts.PostJSON("/api/v1/auth/logout", nil).Body.Close()

	// Login — mfa_required
	resp = ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email":    "mfa-recovery@example.com",
		"password": "SecurePassword123",
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("login: expected 200, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()

	// Use recovery code instead of TOTP
	resp = ts.PostJSON("/api/v1/auth/mfa/recovery", map[string]string{
		"code": firstCode,
	})
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("recovery: expected 200, got %d: %s", resp.StatusCode, body)
	}
	var recoveryResult map[string]interface{}
	ts.DecodeJSON(resp, &recoveryResult)
	if recoveryResult["email"] != "mfa-recovery@example.com" {
		t.Fatalf("recovery: expected email, got %v", recoveryResult["email"])
	}

	// /me should work now
	resp = ts.Get("/api/v1/auth/me")
	if resp.StatusCode != http.StatusOK {
		body := readBody(t, resp)
		t.Fatalf("/me after recovery: expected 200, got %d: %s", resp.StatusCode, body)
	}
	resp.Body.Close()
}
