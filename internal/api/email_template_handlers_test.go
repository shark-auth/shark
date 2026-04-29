package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// seedTemplates runs the storage-layer seed so the /admin/email-templates
// handlers have rows to list/get/patch against. NewTestServer doesn't call
// this path (only the full server binary does), so each handler test opts
// in explicitly.
func seedTemplates(t *testing.T, ts *testutil.TestServer) {
	t.Helper()
	if err := ts.Store.SeedEmailTemplates(context.Background()); err != nil {
		t.Fatalf("SeedEmailTemplates: %v", err)
	}
}

type templateListResp struct {
	Data []*storage.EmailTemplate `json:"data"`
}

func TestListEmailTemplates_Default(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	resp := ts.GetWithAdminKey("/api/v1/admin/email-templates")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out templateListResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Data) != 5 {
		t.Fatalf("expected 5 seeded templates, got %d", len(out.Data))
	}
}

func TestGetEmailTemplate_NotFound(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	resp := ts.GetWithAdminKey("/api/v1/admin/email-templates/does_not_exist")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPatchEmailTemplate_UpdatesSubject(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	resp := ts.PatchJSONWithAdminKey("/api/v1/admin/email-templates/magic_link", map[string]any{
		"subject": "Custom subject for {{.AppName}}",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var patched storage.EmailTemplate
	if err := json.NewDecoder(resp.Body).Decode(&patched); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if patched.Subject != "Custom subject for {{.AppName}}" {
		t.Errorf("subject: want custom, got %q", patched.Subject)
	}
	// Other fields preserved (seed default header_text is "Click to sign in").
	if patched.HeaderText != "Click to sign in" {
		t.Errorf("header_text: want seed default, got %q", patched.HeaderText)
	}

	// Confirm persistence via a fresh GET.
	resp2 := ts.GetWithAdminKey("/api/v1/admin/email-templates/magic_link")
	defer resp2.Body.Close()
	var got storage.EmailTemplate
	if err := json.NewDecoder(resp2.Body).Decode(&got); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if got.Subject != "Custom subject for {{.AppName}}" {
		t.Errorf("re-GET subject: want custom, got %q", got.Subject)
	}
}

type previewResp struct {
	HTML    string `json:"html"`
	Subject string `json:"subject"`
}

func TestPreviewEmailTemplate_DefaultBranding(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/email-templates/magic_link/preview", map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out previewResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.HTML == "" {
		t.Error("expected non-empty html")
	}
	if out.Subject == "" {
		t.Error("expected non-empty subject")
	}
	// Default sample data substitutes {{.AppName}} => "SharkAuth".
	if !strings.Contains(out.Subject, "SharkAuth") {
		t.Errorf("subject: want contains SharkAuth, got %q", out.Subject)
	}
}

func TestPreviewEmailTemplate_OverrideBranding(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/email-templates/magic_link/preview", map[string]any{
		"config": map[string]any{
			"primary_color": "#ff0000",
		},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out previewResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !strings.Contains(out.HTML, "#ff0000") {
		t.Errorf("expected override color #ff0000 in html, not found. HTML=%q", out.HTML)
	}
}

func TestResetEmailTemplate_RestoresSeed(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	// First, mutate.
	patchResp := ts.PatchJSONWithAdminKey("/api/v1/admin/email-templates/magic_link", map[string]any{
		"subject": "Some custom subject",
	})
	patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d", patchResp.StatusCode)
	}

	// Then reset.
	resetResp := ts.PostJSONWithAdminKey("/api/v1/admin/email-templates/magic_link/reset", nil)
	defer resetResp.Body.Close()
	if resetResp.StatusCode != http.StatusOK {
		t.Fatalf("reset: expected 200, got %d", resetResp.StatusCode)
	}
	var got storage.EmailTemplate
	if err := json.NewDecoder(resetResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode reset: %v", err)
	}
	if got.Subject != "Sign in to {{.AppName}}" {
		t.Errorf("reset subject: want seed default, got %q", got.Subject)
	}

	// Confirm persistence via fresh GET.
	getResp := ts.GetWithAdminKey("/api/v1/admin/email-templates/magic_link")
	defer getResp.Body.Close()
	var fresh storage.EmailTemplate
	if err := json.NewDecoder(getResp.Body).Decode(&fresh); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if fresh.Subject != "Sign in to {{.AppName}}" {
		t.Errorf("re-GET subject: want seed default, got %q", fresh.Subject)
	}
}

func TestSendTestEmail_MissingToEmail(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/email-templates/magic_link/send-test", map[string]any{})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSendTestEmail_SendsSuccessfully(t *testing.T) {
	ts := testutil.NewTestServer(t)
	seedTemplates(t, ts)

	beforeCount := ts.EmailSender.MessageCount()
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/email-templates/magic_link/send-test", map[string]any{
		"to_email": "someone@example.com",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["status"] != "sent" {
		t.Errorf("status: want sent, got %v", out["status"])
	}
	if ts.EmailSender.MessageCount() != beforeCount+1 {
		t.Fatalf("expected 1 new message, got %d (before=%d)", ts.EmailSender.MessageCount(), beforeCount)
	}
	msg := ts.EmailSender.LastMessage()
	if msg.To != "someone@example.com" {
		t.Errorf("msg.To: want someone@example.com, got %q", msg.To)
	}
	if msg.Subject == "" {
		t.Error("msg.Subject: expected non-empty")
	}
	if msg.HTML == "" {
		t.Error("msg.HTML: expected non-empty")
	}
}
