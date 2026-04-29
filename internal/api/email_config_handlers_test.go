package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shark-auth/shark/internal/testutil"
)

func TestEmailConfigInviteRedirectURL_RoundTrip(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Fresh install: all fields empty.
	getResp := ts.GetWithAdminKey("/api/v1/admin/email-config")
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET email-config: %d", getResp.StatusCode)
	}
	var initial struct {
		InviteRedirectURL string `json:"invite_redirect_url"`
	}
	ts.DecodeJSON(getResp, &initial)
	if initial.InviteRedirectURL != "" {
		t.Fatalf("expected empty invite_redirect_url on fresh install, got %q", initial.InviteRedirectURL)
	}

	// PATCH with invite_redirect_url.
	want := "https://myapp.com/invite-landing"
	patchResp := ts.PatchJSONWithAdminKey("/api/v1/admin/email-config", map[string]string{
		"invite_redirect_url": want,
	})
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH email-config: %d", patchResp.StatusCode)
	}
	var patched struct {
		InviteRedirectURL string `json:"invite_redirect_url"`
	}
	ts.DecodeJSON(patchResp, &patched)
	if patched.InviteRedirectURL != want {
		t.Errorf("PATCH response: expected %q, got %q", want, patched.InviteRedirectURL)
	}

	// GET again â€” persisted.
	getResp2 := ts.GetWithAdminKey("/api/v1/admin/email-config")
	if getResp2.StatusCode != http.StatusOK {
		t.Fatalf("GET email-config (2nd): %d", getResp2.StatusCode)
	}
	var final struct {
		InviteRedirectURL    string `json:"invite_redirect_url"`
		VerifyRedirectURL    string `json:"verify_redirect_url"`
		ResetRedirectURL     string `json:"reset_redirect_url"`
		MagicLinkRedirectURL string `json:"magic_link_redirect_url"`
	}
	ts.DecodeJSON(getResp2, &final)
	if final.InviteRedirectURL != want {
		t.Errorf("GET after PATCH: expected %q, got %q", want, final.InviteRedirectURL)
	}
	// Other fields unaffected.
	if final.VerifyRedirectURL != "" || final.ResetRedirectURL != "" || final.MagicLinkRedirectURL != "" {
		t.Errorf("unexpected side-effects on other fields: %+v", final)
	}
}

func TestEmailConfigAllFields_RoundTrip(t *testing.T) {
	ts := testutil.NewTestServer(t)

	payload := map[string]string{
		"verify_redirect_url":     "https://app.com/verified",
		"reset_redirect_url":      "https://app.com/reset",
		"magic_link_redirect_url": "https://app.com/dashboard",
		"invite_redirect_url":     "https://app.com/invite-accept",
	}
	patchResp := ts.PatchJSONWithAdminKey("/api/v1/admin/email-config", payload)
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH: %d", patchResp.StatusCode)
	}
	var result map[string]string
	if err := json.NewDecoder(patchResp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for k, v := range payload {
		if result[k] != v {
			t.Errorf("field %q: expected %q, got %q", k, v, result[k])
		}
	}
}
