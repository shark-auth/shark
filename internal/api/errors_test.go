package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestNewErrorAndWithDocsURL(t *testing.T) {
	e := NewError(CodePasswordTooShort, "Password must be at least 12 characters").
		WithDocsURL(CodePasswordTooShort).
		WithDetails(map[string]any{"min_length": 12})

	if e.Error != "password_too_short" {
		t.Fatalf("error = %q", e.Error)
	}
	if e.Code != "password_too_short" {
		t.Fatalf("code = %q", e.Code)
	}
	if e.DocsURL != "https://docs.shark-auth.com/errors/password_too_short" {
		t.Fatalf("docs_url = %q", e.DocsURL)
	}
	if got, ok := e.Details["min_length"].(int); !ok || got != 12 {
		t.Fatalf("details = %v", e.Details)
	}
}

func TestWriteErrorShape(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, 400, NewError(CodeInvalidRequest, "missing email").WithDocsURL(CodeInvalidRequest))

	if got := rr.Code; got != 400 {
		t.Fatalf("status = %d", got)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, key := range []string{"error", "message", "code", "docs_url"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing key %q in body %v", key, body)
		}
	}
	if _, ok := body["details"]; ok {
		t.Errorf("details should be omitted when nil, got %v", body["details"])
	}
}

func TestWriteErrorOmitsEmpty(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, 500, NewError(CodeInternal, "boom"))
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if _, ok := body["docs_url"]; ok {
		t.Errorf("docs_url should be omitted when unset")
	}
}
